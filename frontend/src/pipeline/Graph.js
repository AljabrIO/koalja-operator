import React, { Component } from 'react';
import ReactEcharts from 'echarts-for-react';

class Graph extends Component {
  getOption = () => {
    const spec = this.props.pipeline;
    let yOffsets = {};
    let taskNodes = spec.tasks.map((t, i) => {
      let stats = this.props.taskStats.find(x => x.name === t.name);
      const x = (!hasInputs(t)) ? 100 : (!hasConnectedOutputs(t, spec)) ? 700 : 400;
      const yOffsetsKey = `x${x}`;
      const y = (yOffsets[yOffsetsKey] || 50) + 50;
      yOffsets[yOffsetsKey] = y;
      return {
        name: t.name,
        category: 'task',
        label: {
          show: true,
          position: 'bottom',
          color: 'black'
        },
        itemStyle: {
          color: taskNodeColor(stats),
        },
        symbolSize: 60,
        symbolRotate: 0,
        x: x,
        y: y,
        fixed: true, //(!hasInputs(t)) || (!hasConnectedOutputs(t, spec)),
        task: t,
        stats: stats
      };
    });
    let inputNodes = flatten(taskNodes.map(n => (n.task.inputs || []).map(x => ({
      name: `${n.task.name}/${x.name}`,
      category: 'input',
      x: n.x - n.symbolSize / 2,
      y: n.y,
      fixed: n.fixed,
      symbol: 'roundRect',
      stats: (n.stats.inputs || []).find(s => s.name === x.name),
    }))));
    let outputNodes = flatten(taskNodes.map(n => (n.task.outputs || []).map(x => ({
      name: `${n.task.name}/${x.name}`,
      category: 'output',
      x: n.x + n.symbolSize / 2,
      y: n.y,
      fixed: n.fixed,
      symbol: 'diamond',
      stats: (n.stats.outputs || []).find(s => s.name === x.name),
    }))));
    let nodes = taskNodes.concat(inputNodes, outputNodes);
    //console.log(nodes);

    let taskLinks = (spec.links || []).map(l => {
      let stats = this.props.linkStats.find(x => x.name === l.name);
      return {
        name: `${l.name}`,
        label: {
          show: true,
          formatter: (e) => formatLinkLabel(e.data),
        },
        source: l.sourceRef,
        target: l.destinationRef,
        value: 2,
        lineStyle: {
          curveness: 0.1,
          width: Math.min((stats.annotatedvalues_waiting || 0) + 1, 20),
        },
        symbol: ['none', 'arrow'],
        stats: stats
      };
    });
    let taskInputLinks = flatten(spec.tasks.map(t => {
      const inputs = t.inputs || [];
      return inputs.map(x => ({
        name: `${t.name}/${x.name}`,
        source: `${t.name}/${x.name}`,
        target: t.name,
        value: 1
      }));
    }));
    let taskOutputLinks = flatten(spec.tasks.map(t => (t.outputs || []).map(x => ({
      name: `${t.name}/${x.name}`,
      source: t.name,
      target: `${t.name}/${x.name}`,
      value: 1
    }))));
    let graphLinks = [].concat(taskLinks, taskInputLinks, taskOutputLinks);

    return {
      legend: {
        itemWidth: 15,
        itemHeight: 15,
        data: [{
          name: 'task',
          icon: 'circle'
        }, {
          name: 'input',
          icon: 'roundRect'
         }, {
           name: 'output',
           icon: 'diamond'
         }]
      },
      animationDurationUpdate: 1500,
      animationEasingUpdate: 'quinticInOut',
      backgroundColor: '#efefef',
      tooltip: {
        formatter: '{b}'
      },
      series: [{
        type: 'graph',
        layout: 'force',
        animation: false,
        label: {
          color: 'blue',
          normal: {
            position: 'right',
            formatter: formatLabel,
          }
        },
        tooltip: {
          formatter: formatTooltip,
        },
        roam: true,
        categories: [{
          name: 'task'
        }, {
          name: 'input',
          symbol: 'diamond'
        }, {
          name: 'output',
          symbol: 'pin'
        }],
        data: nodes,
        force: {
          edgeLength: [300, 20, 20],
          repulsion: 50,
          gravity: 0.0
        },
        links: graphLinks,
        lineStyle: {
          color: 'source'
        },
      }]
    };
  };

  onChartClick(e) {
    console.log(e);
  }

  render() {
    let onEvents = {
      'click': this.onChartClick
    //'legendselectchanged': this.onChartLegendselectchanged
    }
    return (
      <ReactEcharts
        option={this.getOption()}
        style={{
          height: '700px',
          width: '100%'
        }}
        onEvents={onEvents}
      />
    );
  }
}

function flatten(a) {
  return Array.isArray(a) ? [].concat(...a.map(flatten)) : a;
}

let hasInputs = (t) => ((t.inputs || []).length > 0)
//let hasOutputs = (t) => ((t.outputs || []).length > 0)
let hasConnectedOutputs = (t, spec) => ((t.outputs || []).some(o => outputIsConnected(t, o, spec)))
let outputIsConnected = (t, output, spec) => (spec.links.some(l => l.sourceRef === `${t.name}/${output.name}`))

let formatLabel = (e) => {
  switch (e.data.category) {
    case "task":
      return formatTaskNodeLabel(e.data);
    default:
      return "";
  }
};
let formatTaskNodeLabel = (n) => {
  const stats = n.stats || {};
  return [
    n.name,
    (stats.snapshots_in_progress > 0) ? `In progress ${stats.snapshots_in_progress}` : undefined,
    (stats.snapshots_waiting > 0) ? `Waiting ${stats.snapshots_waiting}` : undefined,
    `Succeeded ${stats.snapshots_succeeded || 0}`,
    (stats.snapshots_failed > 0) ? `Failed ${stats.snapshots_failed || 0}` : undefined,
  ].filter(x => (typeof x === 'string')).join("\n");
};
let formatLinkLabel = (n) => {
  const stats = n.stats || {};
  return `${stats.annotatedvalues_waiting || 0} / ${stats.annotatedvalues_in_progress || 0} / ${stats.annotatedvalues_acknowledged || 0}`;
};

let formatTooltip = (e) => {
  switch (`${e.dataType}/${e.data.category || ""}`) {
    case "node/task":
      return formatTaskNodeTooltip(e.data);
    case "node/input":
      return formatInputNodeTooltip(e.data);
    case "node/output":
      return formatOutputNodeTooltip(e.data);
    case "edge/":
      return formatLinkTooltip(e.data);
    default:
  }
};
let formatTaskNodeTooltip = (n) => {
  const stats = n.stats || {};
  return [
    `<b>${n.name}</b>`,
    "Executions:",
    (stats.snapshots_in_progress > 0) ? `- In progress ${stats.snapshots_in_progress}` : undefined,
    (stats.snapshots_waiting > 0) ? `- Waiting ${stats.snapshots_waiting}` : undefined,
    `- Succeeded ${stats.snapshots_succeeded || 0}`,
    `- Failed ${stats.snapshots_failed || 0}`,
  ].filter(x => (typeof x === 'string')).join("<br/>");
};
let formatInputNodeTooltip = (n) => {
  const stats = n.stats || {};
  return [
    `<b>${n.name}</b>`,
    "Annotated values:",
    `- Received ${stats.annotatedvalues_received || 0}`,
    `- In progress ${stats.annotatedvalues_in_progress || 0}`,
    `- Processed ${stats.annotatedvalues_processed || 0}`,
    `- Skipped ${stats.annotatedvalues_skipped || 0}`,
  ].filter(x => (typeof x === 'string')).join("<br/>");
};
let formatOutputNodeTooltip = (n) => {
  const stats = n.stats || {};
  return [
    `<b>${n.name}</b>`,
    "Annotated values:",
    `- Published ${stats.annotatedvalues_published || 0}`,
  ].filter(x => (typeof x === 'string')).join("<br/>");
};
let formatLinkTooltip = (n) => {
  const stats = n.stats || {};
  return [
    `<b>${n.name}</b>`,
    "Annotated values:",
    `- Waiting ${stats.annotatedvalues_waiting || 0}`,
    `- In progress ${stats.annotatedvalues_in_progress || 0}`,
    `- Acknowledged ${stats.annotatedvalues_acknowledged || 0}`,
  ].filter(x => (typeof x === 'string')).join("<br/>");
};

let taskNodeColor = (stats) => {
  if (stats.snapshots_failed > 0) return 'red';
  if (stats.snapshots_in_progress > 0) return '#ffcc00';
  if (stats.snapshots_waiting > 0) return 'orange';
  if (stats.snapshots_succeeded > 0) return 'green';
  return 'gray';
};

export default Graph;

