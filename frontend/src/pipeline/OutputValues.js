import React, { Component } from 'react';
import { Header, Table } from 'semantic-ui-react';

const RowView = ({id,data}) => (
  <Table.Row>
    <Table.Cell>{id}</Table.Cell>
    <Table.Cell>{data}</Table.Cell>
  </Table.Row>
);

const TableView = ({values}) => (
  <Table celled>
    <Table.Header>
      <Table.Row>
        <Table.HeaderCell>ID</Table.HeaderCell>
        <Table.HeaderCell>Data</Table.HeaderCell>
      </Table.Row>
    </Table.Header>
    <Table.Body>
      {values.map((value) => <RowView
        key={value.id}
        id={value.id}
        data={value.data}
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

