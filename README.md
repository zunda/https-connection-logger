# https-connection-logger
A Heroku app to log connections to remote servers

## Run as a Heroku app
```
$ heroku create <app-name>
$ git push heroku main
$ heroku addons:add <logging-addon>
$ heroku config:set TARGETS=https://www.example.com/,https://www.example.org/
$ heroku ps:scale worker=1
```
