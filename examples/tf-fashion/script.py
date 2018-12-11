import sys

# TensorFlow and tf.keras
import tensorflow as tf
from tensorflow import keras

# Helper libraries
import numpy as np
import matplotlib.pyplot as plt
from shutil import copyfile
import tempfile

print('Starting script.py')

# Prepare
filepath = sys.argv[1]
print('Train model and write to: ', filepath)

print(tf.__version__)

# Load dataset
fashion_mnist = keras.datasets.fashion_mnist

(train_images, train_labels), (test_images, test_labels) = fashion_mnist.load_data()

# Build the model
model = keras.Sequential([
    keras.layers.Flatten(input_shape=(28, 28)),
    keras.layers.Dense(128, activation=tf.nn.relu),
    keras.layers.Dense(10, activation=tf.nn.softmax)
])

# Compile model
model.compile(optimizer=tf.train.AdamOptimizer(), 
              loss='sparse_categorical_crossentropy',
              metrics=['accuracy'])

# Train model
model.fit(train_images, train_labels, epochs=5)

# Evaluate accuracy
test_loss, test_acc = model.evaluate(test_images, test_labels)

print('Test accuracy:', test_acc)

# Save model
f = tempfile.NamedTemporaryFile(delete=False)
f.close()
tmpFilepath = f.name

tf.keras.models.save_model(
    model,
    tmpFilepath,
    overwrite=True,
    include_optimizer=True
)

copyfile(tmpFilepath, filepath)
