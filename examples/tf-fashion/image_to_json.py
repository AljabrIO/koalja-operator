import sys
import json
from tensorflow.keras.preprocessing.image import img_to_array as img_to_array
from tensorflow.keras.preprocessing.image import load_img as load_img

imagepath = sys.argv[1]
outputpath = sys.argv[2]

image = img_to_array(load_img(imagepath, target_size=(28,28))) / 255.
payload = {
  "instances": [{'input_image': image.tolist()}]
}

with open(outputpath, 'w') as outfile:
    json.dump(payload, outfile)
