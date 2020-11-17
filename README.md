# sponsor-service

A microservice for a fictional application that allows conference organizers to effectively manage their conferences.

This microservice specifically handles sponsorships at a conference.

## Table of Contents

* [Architecture](docs/ARCHITECTURE.md)
* [REST API Documentation](docs/REST_API.md)
* [RabbitMQ Spec](docs/RABBITMQ_MESSAGES.md)
* [Development](docs/DEVELOPMENT.md)
* [How to run this service with Docker a.k.a production](docs/PRODUCTION.md)
* [Helpful Links](docs/LINKS.md)

### To run this service

Follow the steps documented [here](docs/PRODUCTION.md)

### Dependencies

The sponsor service relies on the Event service. While it is totally possible to create events through the sponsor
service REST api, to get the full microservice experience, it is best to run the event service as well.

The sponsor service relies on messages sent to a queue on RabbitMQ.

When events are created, a message is sent to the channel `event.create` that looks exactly like this:
```
EVENT CREATED ::: { "id": 1337, "name":"Event Name", "sponsors":[{ "name": "Platinum", "cost": 14500, "freeBadges": 10 }] }
```

When events are modified, the sponsor service only needs the items to update in it's message. It expects this
message to be sent to the channel: `event.modify`. It must include the event id from the event service, with the key: `id`

An example of a message to modify an event:
```
EVENT MODIFED ::: { "id": 1337, "name":"Changed Event Name", "sponsors":[{ "name": "Gold", "cost": 1000, "freeBadges": 8 }] }
```

With the `event.create` and `event.modify` messages from above. This will result in the sponsor service
translating the event in the anti-corruption layer so it will look like this:
```
{
    "id": 1,
    "eventServiceId": 1337,
    "name": "Changed Event Name",
    "levels": [
        {   
            "name": "Platinum",
            "cost": "14500",
            "maxFreeBadgesPerSponsor": 10,
            "maxSponsors": 0
        },
        {   
            "name": "Gold",
            "cost": "1000",
            "maxFreeBadgesPerSponsor": 8,
            "maxSponsors": 0
        }
    ],
    "sponsors": null
}
```

### Dependents

Because the sponsor service is relied upon by the badges service, any time a sponsor team member is added 
using the sponsor-service REST api, we send a RabbitMQ message to the channel: `sponsor.member.create` with
the shape:
```
{
    "id": 12,
    "name": "Firstname Lastname",
    "email": "first.last@doge.com",
    "eventName": "Changed Event Name",
    "eventId": 1,
    "organization": "Doge Company",
    "sponsorId": 321,
    "sponsorLevel": "Diamond+ Extra"
}
```
This shape is further explained [here](docs/RABBITMQ_MESSAGES.md).