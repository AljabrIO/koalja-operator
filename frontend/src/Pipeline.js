import React, { Component } from 'react';
import ReactEcharts from 'echarts-for-react';
import api, { isUnauthorized } from './api/api';
import ReactTimeout from 'react-timeout';

class Pipeline extends Component {
  state = {
    pipeline: undefined,
    error: undefined
  };

  componentDidMount() {
    this.reloadPipeline();
  }

  reloadPipeline = async() => {
    try {
      const spec = await api.get('/v1/pipeline');
      this.setState({
        pipeline: spec,
        error: undefined
      });
      console.log(spec);
    } catch (e) {
      this.setState({
        error: e.message
      });
      /*if (isUnauthorized(e)) {
        this.props.doLogout();
      }*/
    }
    this.props.setTimeout(this.reloadPipeline, 10000);
  }

  getOption = () => {
    const spec = this.state.pipeline;
    let taskNodes = spec.tasks.map((x, i) => ({
      name: x.name,
      category: 'task',
      label: {
        show: true,
        position: 'bottom'
      },
      symbolSize: 60,
      symbolRotate: 0,
      x: (!hasInputs(x)) ? 100 : (!hasConnectedOutputs(x, spec)) ? 700 : 400,
      y: 100 + i * 50,
      fixed: (!hasInputs(x)) || (!hasConnectedOutputs(x, spec)),
      task: x
    }));
    let inputNodes = flatten(taskNodes.map(n => (n.task.inputs || []).map(x => ({
      name: `${n.task.name}/${x.name}`,
      category: 'input',
      x: n.x - n.symbolSize/2,
      y: n.y,
      fixed: n.fixed
    }))));
    let outputNodes = flatten(taskNodes.map(n => (n.task.outputs || []).map(x => ({
      name: `${n.task.name}/${x.name}`,
      category: 'output',
      x: n.x + n.symbolSize/2,
      y: n.y,
      fixed: n.fixed
    }))));
    let nodes = taskNodes.concat(inputNodes, outputNodes);
    console.log(nodes);

    let taskLinks = spec.links.map(x => ({
      name: x.name,
      label: {
        show: false
      },
      source: x.sourceRef, 
      target: x.destinationRef,
      value: 2,
      lineStyle: {
        curveness: 0.1
      }
    }));
    let taskInputLinks = flatten(spec.tasks.map(t => (t.inputs || []).map(x => ({
      name: `${t.name}/${x.name}`,
      source: `${t.name}/${x.name}`,
      target: t.name,
      value: 1
    }))));
    let taskOutputLinks = flatten(spec.tasks.map(t => (t.outputs || []).map(x => ({
      name: `${t.name}/${x.name}`,
      source: t.name,
      target: `${t.name}/${x.name}`,
      value: 1
    }))));
    let links = [].concat(taskLinks, taskInputLinks, taskOutputLinks);

    return {
      legend: {
        data: ['task', 'input', 'output']
      },
      animationDurationUpdate: 1500,
      animationEasingUpdate: 'quinticInOut',
      tooltip: {
        formatter: '{b}'
      },
      series: [{
        type: 'graph',
        layout: 'force',
        animation: false,
        label: {
          normal: {
            position: 'right',
            formatter: '{b}'
          }
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
        links: links,
        lineStyle: {
          color: 'source'
        }  
      }]
    };
  };

  onChartClick(e) { console.log(e); }

  render() {
    let onEvents = {
      'click': this.onChartClick
      //'legendselectchanged': this.onChartLegendselectchanged
    }
    if (this.state.pipeline) {
      return (
        <ReactEcharts 
          option={this.getOption()} 
          style={{height: '700px', width: '100%'}}
          onEvents={onEvents} 
        />
      );
    }
    return (<div>Loading...</div>);
  }
}

function flatten(a) {
  return Array.isArray(a) ? [].concat(...a.map(flatten)) : a;
}

let hasInputs = (t) => ((t.inputs || []).length > 0)
let hasOutputs = (t) => ((t.outputs || []).length > 0)
let hasConnectedOutputs = (t, spec) => ((t.outputs || []).some(o => outputIsConnected(t, o, spec)))
let outputIsConnected = (t, output, spec) => (spec.links.some(l => l.sourceRef === `${t.name}/${output.name}`))

export default ReactTimeout(Pipeline);

