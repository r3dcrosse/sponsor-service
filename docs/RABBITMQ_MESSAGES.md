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
    "email": "first.last@doge.com",
    "eventName": "JSconf EU",
    "eventId": 123,
    "organization": "Doge Company",
    "sponsorId": 321,
    "sponsorLevel": "Diamond+ Extra"
}
```

### The shape explained
```
{
    "id": 1337, // ID the sponsor service uses to keep track of the member that was created
    "name": "Firstname Lastname",
    "email": "first.last@doge.com",
    "eventName": "JSconf EU",
    "eventId": 123, // Event ID the sponsor service uses to keep track of the event
    "organization": "Doge Company",
    "sponsorId": 321, // Sponsor ID the sponsor service uses to keep track of the sponsoring organization
    "sponsorLevel": "Diamond+ Extra"
}
```