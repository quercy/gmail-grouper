# gmail-grouper
Pulls the Gmail API for messages and lists all of the senders (so you can see who the top offenders are in your inbox).

Written as a small tool meant to be passed into sort, uniq, etc.

## Installing
I don't intend to wrap this up nicely and abstract away the Gmail API, so you'll need to set it up for yourself and download the packages.

### Download the tool

```
go get -u github.com/quercy/gmail-grouper
```

### Enable the Gmail API 
Following the instructions [here](https://developers.google.com/gmail/api/quickstart/go), do steps 1 and 2. Save the `credentials.json` file to wherever the `gmail-grouper` repo is, or pass the location via `gmail-grouper -c ~/path/to/credentials.json`.

### Authorize the app
Run:
```
gmail-grouper -c ~/Downloads/credentials.json
```
You will see a message like "Go to the following link in your browser then type the authorization code: https://...". Follow the instructions.

## Running the tool

```
gmail-grouper --help                                                    
Usage of gmail-grouper:
  -c string
        location of credentials.json (default "credentials.json")
  -l int
        number of messages to retrieve
  -o int
        number of messages to offset
```

Output:
```
gmail-grouper -o 200 -l 1000 -c ~/Downloads/credentials.json | sort | uniq -c | sort -n
...
  13 Patreon <bingo@patreon.com>
  16 Reddit <noreply@redditmail.com>
  17 Wayfair <editor@members.wayfair.com>
  18 Uniqlo USA <email@usa.uniqlo.com>
  19 Flour Bakery <info@flourbakery.com>
  49 Friend 1 <xxx@google.com>
  93 Friend 2 <xxx@gmail.com>
 105 Reid Savage <reid.savage@gmail.com>
```

## Todo
* Smartly group similair senders together
* Cache results of Gmail API call on disk