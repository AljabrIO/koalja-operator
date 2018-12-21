# linecount-mergeinto.yaml

This pipeline shows a variation of the `linecount` pipeline, that demonstrates
the merge capabilities of task inputs.

The pipeline has two inputs that send the files you drop into them into
separate inputs of the `removeDuplicates` task.
The second input of that task uses a `mergeInto` property to merge
all its input into the first task input.

This results in a single stream of document (with distinct lines per document)
coming from two separate input streams.

## Status

This pipeline is fully functional.