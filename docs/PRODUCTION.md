# Running this service as a Docker container

You need Docker installed on your system to do this...

## Step 1: Clone this repo
Make sure to `cd` into the cloned repo
```
git clone https://github.com/r3dcrosse/sponsor-service

cd sponsor-service
```

## Step 2: Build the Docker image
This may take a couple minutes to complete
```
docker build -t sponsor-service .
```
Once this finishes, verify the image was created by running `docker images`. You should get output
that looks like this:
```
🌈  sponsor-service  🚀 (っ◔◡◔)っ[̲̅ docker images
REPOSITORY               TAG                 IMAGE ID            CREATED             SIZE
sponsor-service          latest              4e4a8a0a0af8        5 minutes ago       314MB
```

## Step 3: Run the Docker image
Note: you must know what IP and port rabbitMQ is running on because you
will pass those in as an argument when running the docker image.

In this example, my IP where rabbitMQ is running is: "192.168.1.24:5672"
```
docker run -p 1337:8000 -e RABBITMQ_IP="192.168.1.24:5672" -it sponsor-service
```
Feel free to replace port 1337 with whatever port you want to run this service on