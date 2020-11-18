# Running this service as a Docker container

You need Docker installed on your system to do this...

# Start RabbitMQ
```
docker run -d --hostname mq --name mq -p 5672:5672 rabbitmq:3
```

# Set up postgres
## Get the latest postgres Docker image
```
docker pull postgres
```
## Create a place to store data locally
This is kinda optional, but it's good to do if you want your data to
persist between starting/stopping your postgres docker image.
```
mkdir ${HOME}/postgres-data
```

## Run the postgres docker image
```
docker run -d --name sponsor-postgres -e POSTGRES_PASSWORD=anyPasswordYouWantGoesHere -v ${HOME}/postgres-data/:/var/lib/postgresql/data -p 5432:5432 postgres
```

## Step 1: Clone this repo
Make sure to `cd` into the cloned repo
```
git clone https://github.com/r3dcrosse/sponsor-service

cd sponsor-service
```

## Step 2: Build the sponsor-service Docker image
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

## Step 3: Run the sponsor-service Docker image
Note: you must know what IP and port rabbitMQ is running on because you
will pass those in as an argument when running the docker image.

In this example, my IP where rabbitMQ is running is: "192.168.1.24:5672"

Note: You must also know what IP and port postgres is running on, as well as the postgres user, password, dbname, and if
you want ssl enabled or not...
```
docker run -p 1337:8000 \
  -e RABBITMQ_IP="192.168.1.24:5672" \
  -e PG_IP="192.168.1.24" \
  -e PG_PORT="5432" \
  -e PG_USER="user" \
  -e PG_PASS="PasswordYouUsedGoesHere" \
  -e PG_DB_NAME="postgres" \
  -e PG_SSL="disable" \
  -it sponsor-service
```
Feel free to replace port 1337 with whatever port you want to run this service on

## Optional steps to fill this microservice with data

Note: This kinda does need to be done in this exact order

```
POST /sponsor-service/v1/event
{
  name: "My Event",
}

// JSON Response
{
    "success": true,
    "data": {
        "event": {
            "id": 1,
            "name": "My Event",
            "levels": null,
            "sponsors": null
        }
    }
}
```

Use the ID returned to create a sponsor with a level for an event
```
POST /sponsor-service/v1/event/1/sponsor
{
    "name": "Doge Company",
    "level": {
        "name": "Diamond+",
        "cost": "$250K",
        "maxSponsors": 1,
        "maxFreeBadgesPerSponsor": 25
    }
}

// JSON Response
{
    "success": true,
    "data": {
        "sponsor": {
            "event": "My Event",
            "eventId": 1,
            "name": "Doge Company",
            "level": {
                "eventId": 1,
                "name": "Diamond+",
                "cost": "$250K",
                "maxSponsors": 1,
                "maxFreeBadgesPerSponsor": 25,
                "id": 1
            },
            "members": null,
            "id": 1
        }
    }
}
```
Use the ID returned to create a member
```
POST /sponsor-service/v1/event/1/sponsor/1/member
{
    "name": "Daviddd",
    "email": "david@david.com"
}

// JSON Response
{
    "success": true,
    "data": {
        "member": {
            "name": "Daviddd",
            "email": "david@david.com",
            "id": 1,
            "sponsorId": 1
        }
    }
}
```

At this point, you should have also noticed a rabbitMQ message sent on the channel: `sponsor.member.created`
```
{
    "email":"david@david.com",
    "eventId":1,
    "eventName":"My Event",
    "id":1,
    "name":"Daviddd",
    "organization":"Doge Company",
    "sponsorId":1,
    "sponsorLevel":"Diamond+"
}
```