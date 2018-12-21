# tf-fashion.yaml

This pipeline shows a machine learning pipeline using TensorFlow to
recognize fashion items.

The pipeline has two distinct chains of tasks.
The first chain is used to train the model and (once trained) serve
the model as a local network service.
The second chain is used to make predictions on images of fashion items.

To use the pipeline, first train the model by dropping a [learn_script.py](./tf-fashion/learn_script.py)
into the `scriptInput` task.
This script will be passed to a `learn` task that will build a model with it.
Note that this can take a long time.

Once the `learn` task has finished, it will pass the resulting model to a
`serve` task that uses `tensorflow/serving` to serve a prediction service
as a local network service.

To make a prediction (after the first learning has completed), drop an image
file into the `imageInput` task. This image will be passed to an `imageConvert`
task that preprocesses the image and extracts a matrix from it.
This matrix will be passed to a `predict` task that queries the prediction
API served by the `serve` task and returns the result.

## Status

This pipeline is function from a Koalja point of view, but needs some work
on the Tensorflow side to match the learning script with the script
to preprocess the image and convert it into a matrix.
The preprocessing script [image_to_json.py](./tf-fashion/image_to_json.py)
is build into a docker image. Use `make` in the [`tf-fashion`](./tf-fashion/)
directory to update that image and push it to the registry.
Make sure to update the tag of that image both in the `Makefile` in the `tf-fashion`
directory as well as the executor image of the pipeline YAML file.