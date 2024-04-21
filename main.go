package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"math"
	"math/rand"
	"net/http"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type MetricLineType int

const (
	InstanceMetricLineType MetricLineType = iota + 1
	GroupMeanMetricLineType
	GroupVarianceMetricLineType
)

func (t MetricLineType) String() string {
	switch t {
	case InstanceMetricLineType:
		return "instance"
	case GroupMeanMetricLineType:
		return "group-mean"
	case GroupVarianceMetricLineType:
		return "group-variance"
	}
	return "unknown"
}

type Dashboard struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Rows        []Row  `json:"rows"`
}

func (d Dashboard) Panels() []string {
	panels := make([]string, 0)
	for _, row := range d.Rows {
		for _, panel := range row.Panels {
			panels = append(panels, panel.Id)
		}
	}
	return panels
}

type Row struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Heights     []int   `json:"heights"`
	Widths      []int   `json:"widths"`
	Panels      []Panel `json:"panels"`
}

type Panel struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Units       string `json:"units"`
}

type PanelUpdate struct {
	Id     string            `json:"id"`
	Type   string            `json:"type,omitempty"`
	Group  string            `json:"group,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
	Error  string            `json:"error,omitempty"`
}

type MetricBoardUpdates struct {
	Panel *PanelUpdate `json:"panel,omitempty"`
}

type MetricBoardTimeUpdateCommand struct {
	Start      int64 `json:"start"`
	End        int64 `json:"end"` // 0 if you want to enable streaming until now()
	Resolution int64 `json:"resolution"`
}

type MetricBoardPanelsUpdateCommand struct {
	ActivePanelIds []string `json:"active"`
	ResetPanelIds  []string `json:"reset"`
}

type MetricBoardCommands struct {
	TimeUpdate        *MetricBoardTimeUpdateCommand   `json:"time,omitempty"`
	PanelsUpdate      *MetricBoardPanelsUpdateCommand `json:"panels,omitempty"`
	ConcurrencyUpdate *int                            `json:"concurrency"`
	RefreshUpdate     *int                            `json:"refresh"`
}

type DataSource interface {
	GetMetric(ctx context.Context, panelId string, query MetricQuery, metrics chan<- Metric) error
}

type MetricBoard interface {
	DataSource
	GetDashboard(ctx context.Context, dashboardId string) (Dashboard, error)
	GetPanel(ctx context.Context, panelId string) (Panel, error)
}

const (
	MaxPanelDataPoints = 100_000
)

type MockMetricBoard struct{}

func (m MockMetricBoard) GetDashboard(ctx context.Context, dashboardId string) (Dashboard, error) {
	return Dashboard{
		Id:          dashboardId,
		Title:       "dashboard title",
		Description: "dashboard description",
		Rows: []Row{
			{
				Title:       "row title",
				Description: "row description",
				Heights:     []int{8},
				Widths:      []int{12, 12},
				Panels: []Panel{
					{
						Id:          "panel-id-1",
						Name:        "panel name 1",
						Description: "panel description 1",
						Units:       "ms",
					},
					{
						Id:          "panel-id-2",
						Name:        "panel name 2",
						Description: "panel description 2",
						Units:       "%",
					},
				},
			},
		},
	}, nil
}

func (m MockMetricBoard) GetPanel(ctx context.Context, panelId string) (Panel, error) {
	return Panel{
		Id:          panelId,
		Name:        "panel name 1",
		Description: "panel description 1",
		Units:       "ms",
	}, nil
}

func (m MockMetricBoard) GetMetric(ctx context.Context, panelId string, query MetricQuery, metrics chan<- Metric) error {
	resolution := query.Resolution.Microseconds()
	start := query.StartTime.UnixMicro() + (resolution-query.StartTime.UnixMicro())%resolution
	end := query.EndTime.UnixMicro() - query.EndTime.UnixMicro()%resolution
	timestamps, values := make([]uint64, 0), make([]float32, 0)
	for start <= end {
		f := fnv.New64()
		_, _ = f.Write([]byte(panelId))
		_, _ = f.Write(binary.LittleEndian.AppendUint64(nil, uint64(start)))
		r := rand.New(rand.NewSource(int64(f.Sum64())))

		timestamps = append(timestamps, uint64(start))
		minute := time.Minute.Microseconds()
		values = append(values, float32(math.Sin(2*math.Pi*float64(start%minute)/float64(minute))+r.Float64()*0.1))
		start += resolution
	}
	metrics <- Metric{
		PanelId:    panelId,
		Type:       InstanceMetricLineType,
		Timestamps: timestamps,
		Values:     values,
	}
	return nil
}

var (
	metricboardLocal = EnvTryParseBool("METRICBOARD_LOCAL")
)

func main() {
	var metricBoard MetricBoard = MockMetricBoard{}

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		Logger.Infof("start http request processing: uri=%v", request.RequestURI)
		path := request.URL.Path
		entityId := request.URL.Query().Get("id")

		if path != "/dashboard" && path != "/panel" {
			Logger.Errorf("unexpected path '%v'", path)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}

		Logger.Infof("requesting ws for path=%v, entity=%v", path, entityId)

		c, err := websocket.Accept(writer, request, &websocket.AcceptOptions{InsecureSkipVerify: metricboardLocal})
		if err != nil {
			Logger.Errorf("failed to accept websocket connection: uri=%v, err=%v", request.RequestURI, err)
			return
		}

		var panels []string
		if path == "/dashboard" {
			dashboard, err := metricBoard.GetDashboard(request.Context(), entityId)
			if err != nil {
				Logger.Errorf("unable to fetch dashboard details: id=%v, err=%v", entityId, err)
				_ = c.Close(http.StatusInternalServerError, "unable to fetch dashboard details")
				return
			}
			dashboardBytes, err := json.Marshal(dashboard)
			if err != nil {
				Logger.Errorf("unable to serialize dashboard details: id=%v, err=%v", entityId, err)
				_ = c.Close(http.StatusInternalServerError, "unable to serialize dashboard details")
				return
			}
			if err = c.Write(request.Context(), websocket.MessageText, dashboardBytes); err != nil {
				_ = c.Close(http.StatusInternalServerError, "failed to write dashboard details")
				return
			}
			panels = dashboard.Panels()
		} else if path == "/panel" {
			panel, err := metricBoard.GetPanel(request.Context(), entityId)
			if err != nil {
				Logger.Errorf("unable to fetch panel details: id=%v, err=%v", entityId, err)
				_ = c.Close(http.StatusInternalServerError, "unable to fetch dashboard details")
				return
			}
			panelBytes, err := json.Marshal(panel)
			if err != nil {
				Logger.Errorf("unable to serialize panel details: id=%v, err=%v", entityId, err)
				_ = c.Close(http.StatusInternalServerError, "unable to serialize dashboard details")
				return
			}
			if err = c.Write(request.Context(), websocket.MessageText, panelBytes); err != nil {
				_ = c.Close(http.StatusInternalServerError, "failed to write dashboard details")
				return
			}
			panels = []string{entityId}
		}

		ctx, cancel := context.WithCancel(request.Context())
		defer cancel()

		commands := NewStreamingReader[MetricBoardCommands](ctx, 0, func() (MetricBoardCommands, error) {
			var command MetricBoardCommands
			err := wsjson.Read(ctx, c, &command)
			return command, err
		})
		results := NewStreamingWriter[MetricResult](ctx, 0, func(result MetricResult) {
			if result.Err != nil {
				update := &MetricBoardUpdates{Panel: &PanelUpdate{Id: result.PanelId, Error: result.Err.Error()}}
				updateBytes, _ := json.Marshal(update)
				_ = c.Write(ctx, websocket.MessageText, updateBytes)
			} else {
				update := &MetricBoardUpdates{Panel: &PanelUpdate{
					Id:     result.PanelId,
					Type:   result.Metric.Type.String(),
					Group:  result.Metric.Group,
					Labels: result.Metric.Labels,
				}}
				updateBytes, _ := json.Marshal(update)
				_ = c.Write(ctx, websocket.MessageText, updateBytes)
				_ = c.Write(ctx, websocket.MessageBinary, EncodeU64(result.Metric.Timestamps))
				_ = c.Write(ctx, websocket.MessageBinary, EncodeF32(result.Metric.Values))
			}
		})
		SubscribeToPanels(ctx, metricBoard, panels, commands, results)

		defer func() {
			Logger.Infof("finish http request processing: uri=%v", request.RequestURI)
		}()
	})
	err := http.ListenAndServe(":8000", handler)
	if err != nil {
		Logger.Errorf("server exited with error: %v", err)
	}
}
