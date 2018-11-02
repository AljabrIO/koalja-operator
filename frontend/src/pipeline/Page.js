import React, { Component } from 'react';
import api from '../api/api';
import ReactTimeout from 'react-timeout';
import Graph from './Graph';
import Websocket from 'react-websocket';

class Page extends Component {
  state = {
    pipeline: undefined,
    linkStats: undefined,
    taskStats: undefined,
    error: undefined
  };

  componentDidMount() {
    this.reloadPipeline();
    this.reloadLinkStats();
    this.reloadTaskStats();
  }

  reloadPipeline = async() => {
    try {
      const spec = await api.get('/v1/pipeline');
      this.setState({
        pipeline: spec,
        error: undefined
      });
    //console.log(spec);
    } catch (e) {
      this.setState({
        error: e.message
      });
    /*if (isUnauthorized(e)) {
      this.props.doLogout();
    }*/
    }
  }

  reloadLinkStats = async() => {
    try {
      const stats = await api.post('/v1/statistics/links', {});
      this.setState({
        linkStats: stats,
        error: undefined
      });
    //console.log(stats);
    } catch (e) {
      this.setState({
        error: e.message
      });
    /*if (isUnauthorized(e)) {
      this.props.doLogout();
    }*/
    }
  }

  reloadTaskStats = async() => {
    try {
      const stats = await api.post('/v1/statistics/tasks', {});
      this.setState({
        taskStats: stats,
        error: undefined
      });
    //console.log(stats);
    } catch (e) {
      this.setState({
        error: e.message
      });
    /*if (isUnauthorized(e)) {
      this.props.doLogout();
    }*/
    }
  }

  handleUpdate = (data) => {
    this.reloadPipeline();
    this.reloadLinkStats();
    this.reloadTaskStats();

    //let result = JSON.parse(data);
    //this.setState({count: this.state.count + result.movement});
  }

  render() {
    let ws = (<Websocket 
      key="socket"
      url={`ws://${window.location.host}/v1/updates`}
      onMessage={this.handleUpdate}/>);

    if (this.state.pipeline && this.state.linkStats && this.state.taskStats) {
      return [ws, (
        <Graph
          key="graph"
          pipeline={this.state.pipeline}
          linkStats={this.state.linkStats.statistics || []}
          taskStats={this.state.taskStats.statistics || []}
        />
        )];
    }
    return [ws, (<div key="loading">Loading...</div>)];
  }
}

export default ReactTimeout(Page);

