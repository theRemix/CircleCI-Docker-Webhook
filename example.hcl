# (required) the url path for quay to request, consider this secret
webhookPath = "CHANGE_ME__DO_NOT_ACTUALLY_USE_THIS_VALUE__SEE_README"

slack {
  # (required) supply webhook url
  webhookUrl = "https://hooks.slack.com/services/TTTTTTTTT/BBBBBBBBB/GGGGGGGGGGGGGGGGGGGGGGGG"

  # (optional) customize
  username  = "Deploy-Bot"     # can be anything
  channel   = "#notifications" # can be any #channel or @user
  iconEmoji = ":whale:"
}

service "svc1-staging" {
  repository = "mynamespace/svc-1"
  conditions = "refs/heads/master"
  cmd        = "docker run --rm --name svc1-staging alpine echo 'deployed svc1-staging:latest'"
  # default deployMessage = "Successfully deployed svc1-staging from refs/tags/master"
}

service "svc1-production" {
  repository = "mynamespace/svc-1"
  conditions = "refs/tags/(.+)"
  cmd        = "docker run --rm --name svc1-production alpine echo 'deployed svc1-production:$1'"

  # (optional)
  deployMessage = "Successfully deployed svc1-production version $1"
}
