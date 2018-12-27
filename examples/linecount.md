# linecount.yaml

This pipeline shows a very simple pipeline, that counts the number
of lines of files that you pass to it.

The pipeline starts with a `FileDrop` task used to receive incoming files.

The next task takes a single file and pipes it through `uniq` to remove
all duplicate lines.

Finally the distinct lines are counted with `wc -l` resulting in a single number.

## Usage

- Deploy the pipeline into a Kubernetes cluster prepared for Koalja:

```bash
kubectl apply -f linecount.yaml
```

- View pipeline UI:

```bash
# Open in browser: http://linecount.default.<domain>
```

- Send a file into the `FileDrop` input task:

```bash
curl -X POST http://linecount.default.<domain>/fileDropInput '@<nameOfFileToSend>'
```

## Status

This pipeline is fully functional.