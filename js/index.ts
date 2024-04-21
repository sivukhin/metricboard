interface Panel {
    name: string
    description: string
    timestamps: number[]
    values: number[]
}

interface MetricBoard {
    getPanel(id: string): Panel

    setRange(start: number, end: number, resolution: number): void

    setConcurrency(concurrency: number): void

    setRefresh(refresh: number): void

    setActivePanels(ids: string[]): void

    resetPanels(ids: string[]): void
}

const F32Bin = 1
const U64Bin = 2

function getUint64(dataView: DataView): number {
    const low = dataView.getUint32(0, true);
    const high = dataView.getUint32(4, true);
    return low + high * (2 ** 32);
}

var newPanel = function (host: string, panelId: string): MetricBoard {
    const metricboard = new WebSocket(`ws://${host}/panel?id=${panelId}`);
    metricboard.addEventListener("open", (event) => {
        console.log(`metricboard for ${panelId} opened`)
    });
    metricboard.addEventListener("close", (event) => {
        console.log(`metricboard for ${panelId} closed`)
    });
    metricboard.addEventListener("message", (event) => {
        if (event.data instanceof Blob) {
            event.data.arrayBuffer().then(buffer => {
                const version = new DataView(buffer.slice(0, 1)).getUint8(0);
                let length, values;
                switch (version) {
                    case F32Bin:
                        length = new DataView(buffer.slice(1, 5)).getUint32(0, true);
                        values = [];
                        for (let i = 0; i < length; i++) {
                            const value = new DataView(buffer.slice(5 + 4 * i, 5 + 4 * (i + 1))).getFloat32(0, true)
                            values.push(value);
                        }
                        break
                    case U64Bin:
                        length = new DataView(buffer.slice(1, 5)).getUint32(0, true);
                        values = [];
                        for (let i = 0; i < length; i++) {
                            const value = getUint64(new DataView(buffer.slice(5 + 8 * i, 5 + 8 * (i + 1))))
                            values.push(value);
                        }
                }
                console.info(length, values)
            })
        }
    });
    return {
        getPanel(id: string): Panel {
            return null;
        },
        resetPanels(ids: string[]) {
            metricboard.send(JSON.stringify({"panels": {"reset": ids}}));
        },
        setActivePanels(ids: string[]) {
            metricboard.send(JSON.stringify({"panels": {"active": ids}}));
        },
        setConcurrency(concurrency: number) {
            metricboard.send(JSON.stringify({"concurrency": concurrency}));
        },
        setRefresh(refresh: number) {
            metricboard.send(JSON.stringify({"refresh": refresh}));
        },
        setRange(start: number, end: number, resolution: number) {
            metricboard.send(JSON.stringify({"time": {"start": start, "end": end, "resolution": resolution}}));
        }
    };
}