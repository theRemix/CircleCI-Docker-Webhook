# CircleCI Docker Webhook

Script to autodeploy new versions of a repo after the docker image has been built in Quay, with optional slack notifications.

Successful CircleCI tests -> triggers Quay Builds
Successful Quay builds -> triggers Deploy Webhook
Successful Deployments -> (optionally) notifies via slack

This repo contains:

  - [sample CircleCI config](./circle.example.yml)
  - CircleCI trigger [Quay build script](./trigger_quay.sh)
  - sample Quay webhook payloads [./test-master-payload.json](refs/heads/master) [./test-tag-payload.json](refs/tags/tag)
  - Deploy Webhook source
  - Deploy Webhook [./example.hcl](./example.hcl)

## Requirements

- `go` to compile webhook
- your deployment server must have `docker` installed and running

## Setup Overview

1. [Quay Setup](#Quay Setup)
1. [Copy the Trigger URL](#Copy the webhook url with token)
1. [CircleCI Setup](#CircleCI Setup)
1. [CircleCI Setup](#CircleCI Setup)


## Quay Setup

<details><summary><h3>New Quay Repo</h3></summary>
<div>

- Create new Container Image Repository
- Link to Custom Git Repository Push
    - see [https://docs.quay.io/guides/custom-trigger.html](https://docs.quay.io/guides/custom-trigger.html)


</div>
</details>

<details><summary><h3>Existing Quay Repo</h3></summary>
<div>

- add Build Trigger
    - see [https://docs.quay.io/guides/custom-trigger.html](https://docs.quay.io/guides/custom-trigger.html)

</div>
</details>

### Copy the webhook url with token

it will look something like this

```
https://$token:T79QKPYYN7BEEFQ2EAXKLLURGEDEADC0F10KAIPINCBTJQV015DSME4787I7OOXK@quay.io/webhooks/push/trigger/17771773-1f33-4f33-a7ee-be870d11d1d1
```

### Create Repository Notification

Go to repo settings

Create Notification

Set : "Dockerfile Build Successfully Completed"

Leave "matching refs" blank

Then issue a notification : "Webhook POST"

Set the Webhook URL to your deployment server

**Optional Slack Notifications**

Create Slack Notifications for other events too.



## CircleCI Setup

Create or Edit the Job

Under Job Settings, edit `Environment Variables` under Build Settings

Add `TRIGGER_URL`

Set the value to the webhook url from quay, escape the `$` with `\$`

```
https://\$token:T79QKPYYN7BEEFQ2EAXKLLURGEDEADC0F10KAIPINCBTJQV015DSME4787I7OOXK@quay.io/webhooks/push/trigger/17771773-1f33-4f33-a7ee-be870d11d1d1
```

## Webhook Setup

### Webhook Configuration

see [./example.hcl](./example.hcl)

```sh
cp example.hcl config.hcl
$EDITOR config.hcl
```

<details><summary><h4>Slack Notifications</h4></summary>
<div>

(Optional)

If you want slack notifications, update the `slack` block in `config.hcl`

otherwise, delete the `slack` block

</div>
</details>


### Compile and upload

Compile the webhook with `make` (or `make linux64` if you are not compiling on linux)

scp `./bin/webhook` and `./config.hcl` up to your server.


### Run webhook

ssh into your server

**Quick and Dirty**

```sh
nohup ./webhook config.hcl >> webhook.log 2>&1 &
```

will start listening for webhook requests on port `2000`

**set the port**

```sh
PORT=2121 nohup ./webhook config.hcl >> webhook.log 2>&1 &
```

**enable extra debug messages**

```sh
DEBUG=1 nohup ./webhook config.hcl >> webhook.log 2>&1 &
```


## Test trigger

update the following three fields in [./test-tag-payload.json](./test-tag-payload.json) to real values (all others are ignored)

- `repository`
- `name`
- `trigger_metadata.ref`

```sh
curl -X POST --data-binary "@./test-tag-payload.json" https://webhook.yourdomain.com/
```
