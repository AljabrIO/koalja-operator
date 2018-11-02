import React, { Component } from 'react';
import api from '../api/api';
import ReactTimeout from 'react-timeout';
import Graph from './Graph';

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
    this.props.setTimeout(this.reloadPipeline, 10000);
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
    this.props.setTimeout(this.reloadLinkStats, 2500);
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
    this.props.setTimeout(this.reloadTaskStats, 2500);
  }

  render() {
    if (this.state.pipeline && this.state.linkStats && this.state.taskStats) {
      return (
        <Graph
          pipeline={this.state.pipeline}
          linkStats={this.state.linkStats.statistics || []}
          taskStats={this.state.taskStats.statistics || []}
        />
        );
    }
    return (<div>Loading...</div>);
  }
}

export default ReactTimeout(Page);

