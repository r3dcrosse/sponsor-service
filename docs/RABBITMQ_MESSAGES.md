# RabbitMQ Spec

Whenever a person is added to a sponsorship team, this service publishes a message
using the channel name: 
```
sponsor.member.created
```

## Example
I add a new member to a sponsor using:
```
POST /sponsor-service/v1/event/1/sponsor/1/member
{ "name": "Firstname Lastname", "email": "first.last@doge.com" }
```
This will publish a JSON message to the channel: `sponsor.member.created` with this shape:
```
{
    "id": 1337,
    "name": "Firstname Lastname",
    "email": "first.last@doge.com"
}
```

### The shape explained
```
{
    "id": 1337, // The ID the sponsor service uses to identify a member
    "name": "Firstname Lastname", // Their name
    "email": "first.last@doge.com" // Their email
}
```