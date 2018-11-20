import React, { Component } from 'react';
import { Header, Table } from 'semantic-ui-react';
import moment from 'moment';

const RowView = ({id,output,data,created_at}) => (
  <Table.Row>
    <Table.Cell>{id}</Table.Cell>
    <Table.Cell>{output}</Table.Cell>
    <Table.Cell>{data}</Table.Cell>
    <Table.Cell>{moment(created_at).fromNow()}</Table.Cell>
  </Table.Row>
);

const TableView = ({values}) => (
  <Table celled>
    <Table.Header>
      <Table.Row>
        <Table.HeaderCell>ID</Table.HeaderCell>
        <Table.HeaderCell>Output</Table.HeaderCell>
        <Table.HeaderCell>Data</Table.HeaderCell>
        <Table.HeaderCell>When</Table.HeaderCell>
      </Table.Row>
    </Table.Header>
    <Table.Body>
      {values.map((value) => <RowView
        key={value.id}
        id={value.id}
        output={value.source_task_output}
        data={value.data}
        created_at={value.created_at}
      />)}
    </Table.Body>
  </Table>
);

class OutputValues extends Component {

  render() {
    let taskName = this.props.taskName;
    console.log(this.props);
    if (taskName) {
      return (
        <div>
          <Header>{taskName}</Header>
          <TableView values={this.props.outputs || []}/>
        </div>
      );
    }
    return (<div>No task selected</div>)
  }
}

export default OutputValues;

