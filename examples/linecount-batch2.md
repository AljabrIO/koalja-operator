# linecount-batch2.yaml

This pipeline shows a variation of the `linecount` pipeline, that uses
batching on its inputs.

Instead of taking distinct lines from individual files, it batches
two input files and takes distinct lines from that batch.

## Usage

- Deploy the pipeline into a Kubernetes cluster prepared for Koalja:

```bash
kubectl apply -f linecount-batch2.yaml
```

- View pipeline UI:

```bash
# Open in browser: http://linecount-batch2.default.<domain>
```

- Send a file into the `FileDrop` input task:

```bash
curl -X POST http://linecount-batch2.default.<domain>/fileDropInput '@<nameOfFileToSend>'
```

## Status

This pipeline is fully functional.