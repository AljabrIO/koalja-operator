# linecount-mergeinto.yaml

This pipeline shows a variation of the `linecount` pipeline, that demonstrates
the merge capabilities of task inputs.

The pipeline has two inputs that send the files you drop into them into
separate inputs of the `removeDuplicates` task.
The second input of that task uses a `mergeInto` property to merge
all its input into the first task input.

This results in a single stream of document (with distinct lines per document)
coming from two separate input streams.

## Usage

- Deploy the pipeline into a Kubernetes cluster prepared for Koalja:

```bash
kubectl apply -f linecount-mergeinto.yaml
```

- View pipeline UI:

```bash
# Open in browser: http://linecount-mergeinto.default.<domain>
```

- Send a file into the `FileDrop` input task:

```bash
curl -X POST http://linecount-mergeinto.default.<domain>/fileDropInput1 '@<nameOfFileToSend>'
# or
curl -X POST http://linecount-mergeinto.default.<domain>/fileDropInput2 '@<nameOfFileToSend>'
```

## Status

This pipeline is fully functional.