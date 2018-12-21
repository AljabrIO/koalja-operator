# linecount.yaml

This pipeline shows a very simple pipeline, that counts the number
of lines of files that you pass to it.

The pipeline starts with a `FileDrop` task used to receive incoming files.

The next task takes a single file and pipes it through `uniq` to remove
all duplicate lines.

Finally the distinct lines are counted with `wc -l` resulting in a single number.

## Status

This pipeline is fully functional.