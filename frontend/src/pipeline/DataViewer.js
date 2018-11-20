import React, { Component } from 'react';
import { Button, Embed, Modal } from 'semantic-ui-react';
import api from '../api/api';
import contentType from 'content-type';

let isText = (x) => {
  const ct = contentType.parse(x || "");
  return ct.type.startsWith("text/");
};

class DataViewer extends Component {
  state = {
    content: undefined,
    contentType: undefined,
    error: undefined
  };

  componentDidMount() {
    this.loadDataView();
  }

  loadDataView = async() => {
    let data = this.props.data;
    if (data) {
      try {
        const resp = await api.postRawResult('/v1/data/view', {
          data: this.props.data
        });
        const contentType = resp.headers.get('Content-Type');
        if (isText(contentType)) {
          const text = await resp.text();
          this.setState({
            content: text,
            contentType: contentType,
            error: undefined
          });
        } else {
          const blob = await resp.blob();
          this.setState({
            content: blob,
            contentType: contentType,
            error: undefined
          });
        }
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
  }

  render() {
    let data = this.props.data;
    if (!data) {
      return (<div>empty...</div>)
    }
    let content = this.state.content;
    if (content) {
      return (
        <Modal trigger={<Button basic small="true" floated="right" circular icon="eye"/>}>
          <Modal.Header>Content</Modal.Header>
          <Modal.Content scrolling>
            <Embed
              active={true}
              content={this.state.content}
            />
          </Modal.Content>
        </Modal>
      );
    }
    return (<div>Loading...</div>)
  }
}

export default DataViewer;

