# tf-fashion.yaml

This pipeline shows a machine learning pipeline using TensorFlow to
recognize fashion items.

This example differs from most other examples. It uses an (implicit) network
link between 2 tasks (`serve` & `predict`).

The pipeline has two distinct chains of tasks.
The first chain is used to train the model and (once trained) serve
the model as a local network service.
The second chain is used to make predictions on images of fashion items.

To use the pipeline, first train the model by dropping a [learn_script.py](./tf-fashion/learn_script.py)
into the `scriptInput` task.
This script will be passed to a `learn` task that will build a model with it.
Note that this can take a long time (at least several minutes).

Once the `learn` task has finished, it will pass the resulting model to a
`serve` task that uses `tensorflow/serving` to serve a prediction service
as a local network service.

To make a prediction (after the first learning has completed), drop an image
file into the `imageInput` task. This image will be passed to an `imageConvert`
task that preprocesses the image and extracts a matrix from it.
This matrix will be passed to a `predict` task that queries the prediction
API served by the `serve` task and returns the result.

## Usage

- Deploy the pipeline into a Kubernetes cluster prepared for Koalja:

```bash
kubectl apply -f tf-fashion.yaml
```

- View pipeline UI:

```bash
# Open in browser: http://tf-fashion.default.<domain>
```

- Send the training script into `FileDrop` input task:

```bash
curl -X POST http://tf-fashion.default.<domain>/scriptInput '@tf-fashion/learn_script.py'
```

- Wait for the learning to complete. It has completed when the `serve`
  task is activated (turns yellow in UI).

- Send the prediction image into `FileDrop` input task:

```bash
curl -X POST http://tf-fashion.default.<domain>/imageInput '@tf-fashion/test/test1.png'
```

## Status

This pipeline is function from a Koalja point of view, but needs some work
on the Tensorflow side to match the learning script with the script
to preprocess the image and convert it into a matrix.
The preprocessing script [image_to_json.py](./tf-fashion/image_to_json.py)
is build into a docker image. Use `make` in the [`tf-fashion`](./tf-fashion/)
directory to update that image and push it to the registry.
Make sure to update the tag of that image both in the `Makefile` in the `tf-fashion`
directory as well as the executor image of the pipeline YAML file.

## TensorFlow issues

### Scripts

While this example is fully functional from a Koalja point of view, the Tensorflow
scripts are not optimal and probably incorrect.

To solve that the code in `learn_script.py` and `image_to_json.py` has to be
altered such that the model created, trained and save in `learn_script.py` matches
with the image matrix that is being build in `image_to_json.py`.
Every time `image_to_json.py` is altered, a new docker image has to be build (run `make`)
and the image task in the pipeline has to be updated accordingly.

### Docker images

The Tensorflow docker images used in this pipeline make use of the AVX instruction set.
While this instruction set is available in almost all modern CPU's, it is not
in a standard KVM that uses qemu. As a result (when running on such a KVM virtual machine)
the Tensorflow docker containers will crash without writing anything in the stdout/err.

To solve this, configure the KVM virtual machine to use the host processor.
