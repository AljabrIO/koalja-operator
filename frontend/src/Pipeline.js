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
    return {
      legend: {
        data: ['HTMLElement', 'WebGL', 'SVG', 'CSS', 'Other']
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
        draggable: true,
        data: spec.tasks.map((x, i) => ({
          name: x.name,
          //x: i * 50,
          //y: 100,
          fixed: false,
          label: {
            show: true
          },
          symbolSize: 60
        })),
        //categories: webkitDep.categories,
        force: {
          // initLayout: 'circular'
          // repulsion: 20,
          edgeLength: 300,
          repulsion: 50,
          gravity: 0.0
        },
        links: spec.links.map(x => ({
          name: x.name,
          label: {
            show: true
          },
          source: x.sourceRef.split('/')[0], 
          target: x.destinationRef.split('/')[0]
        }))
      }]
    };
  };

  render() {
    if (this.state.pipeline) {
      return (
        <ReactEcharts 
          option={this.getOption()} 
          style={{height: '700px', width: '100%'}}
        />
      );
    }
    return (<div>Loading...</div>);
  }
}

export default ReactTimeout(Pipeline);
