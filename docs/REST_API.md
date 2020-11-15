# REST API

## GET /sponsor-service/v1/events
Returns all events the sponsor service knows about.
```
GET /sponsor-service/v1/events

// Example JSON response:
{
  "success": true,
  "data": {
    "events": [
        { "id": 1, "name": "DEFCON" },
        { "id": 2, "name": "kubecon" },
        { "id": 3, "name": ".conf" },
        { "id": 4, "name": "JSconf EU" }
      ]
  } 
}
```

## POST /sponsor-service/v1/event
Creates an event
```
POST /sponsor-service/v1/event
{
  name: "DEFCON",
}

// Example JSON response:
{
  "success": true,
  "data": {
    "event": { "id": 1, "name": "DEFCON" }
  } 
}
```


## GET /sponsor-service/v1/event/{event_id}
Returns a list of sponsor organization names and each sponsor's level for an event.

You must pass an event_id that exists in the sponsor service, otherwise you'll get an error.
```
// Example 1
GET /sponsor-service/v1/event/1

// JSON response:
{
  "success": true,
  "data": {
    "sponsors": [
        { "id": 1, name": "doge company", "level": "Diamond" },
        { "id": 2, "name": "lolcat organization", "level": "Silver+" },
        { "id": 3, "name": "yoooo", "level": "Blue" }
      ]
  }
}

// Example 2 (getting an event ID that does not exist)
GET /sponsor-service/v1/event/1337

// JSON response:
{
  "success": false,
  "error": { 
    "message": "Event does not exist"
  }
}
```

## POST /sponsor-service/v1/event/{event_id}/level
Creates a specific sponsorship level for an event

You must pass an event_id that exists in the sponsor service, otherwise you'll get an error.
```
// Example 1
POST /sponsor-service/v1/event/1/level
{
  "name": "Diamond",
  "cost": "$250K",
  "maxSponsors": 1,
  "maxFreeBadgesPerSponsor": 25
}

// JSON response:
{
  "success": true,
  "data": {
    "level": {
      "eventId": 1,
      "name": "Diamond",
      "cost": "$250K",
      "maxSponsors": 1,
      "maxFreeBadgesPerSponsor": 25
    }
  }
}

// Example 2 (getting an event ID that does not exist)
GET /sponsor-service/v1/event/1337

// JSON response:
{
  "success": false,
  "error": { 
    "message": "Event does not exist"
  }
}
```

## POST /sponsor-service/v1/event/{event_id}/sponsor
Creates a sponsor at a specific level, for a particular event id

You must pass an event_id that exists in the sponsor service, otherwise you'll get an error.
```
// Example 1
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

// JSON response:
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

## POST /sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member
Creates a member for a specific sponsor

```
// Example 1
POST /sponsor-service/v1/event/1/sponsor/1/member
{ "name": "Firstname Lastname", "email": "first.last@doge.com" }

// JSON response:
{
  "success": true,
  "data": {
    "member": {
      "id": 1,
      "name": "Firstname Lastname",
      "email": "first.last@doge.com"
    }
  }
}
```

## DELETE /sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member/{member_id}
Removes a specific member from a sponsor

```
// Example 1
DELETE /sponsor-service/v1/event/{event_id}/sponsor/{sponsor_id}/member/1

// JSON response:
{
  "success": true,
  "message": "Removed this member from the sponsor team",
  "data": {
    "member": {
      "id": 1,
      "name": "Firstname Lastname",
      "email": "first.last@doge.com"
    }
  }
}
```