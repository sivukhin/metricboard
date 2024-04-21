let getUint64 = function(dataView) {
  const low = dataView.getUint32(0, true);
  const high = dataView.getUint32(4, true);
  return low + high * 4294967296;
};
const F32Bin = 1;
const U64Bin = 2;
var newPanel = function(host, panelId) {
  const metricboard = new WebSocket(`ws://${host}/panel?id=${panelId}`);
  metricboard.addEventListener("open", (event) => {
    console.log(`metricboard for ${panelId} opened`);
  });
  metricboard.addEventListener("close", (event) => {
    console.log(`metricboard for ${panelId} closed`);
  });
  metricboard.addEventListener("message", (event) => {
    if (event.data instanceof Blob) {
      event.data.arrayBuffer().then((buffer) => {
        const version = new DataView(buffer.slice(0, 1)).getUint8(0);
        let length, values;
        switch (version) {
          case F32Bin:
            length = new DataView(buffer.slice(1, 5)).getUint32(0, true);
            values = [];
            for (let i = 0;i < length; i++) {
              const value = new DataView(buffer.slice(5 + 4 * i, 5 + 4 * (i + 1))).getFloat32(0, true);
              values.push(value);
            }
            break;
          case U64Bin:
            length = new DataView(buffer.slice(1, 5)).getUint32(0, true);
            values = [];
            for (let i = 0;i < length; i++) {
              const value = getUint64(new DataView(buffer.slice(5 + 8 * i, 5 + 8 * (i + 1))));
              values.push(value);
            }
        }
        console.info(length, values);
      });
    }
  });
  return {
    getPanel(id) {
      return null;
    },
    resetPanels(ids) {
      metricboard.send(JSON.stringify({ panels: { reset: ids } }));
    },
    setActivePanels(ids) {
      metricboard.send(JSON.stringify({ panels: { active: ids } }));
    },
    setConcurrency(concurrency) {
      metricboard.send(JSON.stringify({ concurrency }));
    },
    setRefresh(refresh) {
      metricboard.send(JSON.stringify({ refresh }));
    },
    setRange(start, end, resolution) {
      metricboard.send(JSON.stringify({ time: { start, end, resolution } }));
    }
  };
};
