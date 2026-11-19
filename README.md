# https-connection-logger
A Heroku app to log connections to remote servers

## Run as a Heroku app
```
$ heroku create <app-name>
$ git push heroku main
$ heroku addons:add <logging-addon>
$ heroku config:set TARGET=https://www.example.com/
$ heroku ps:scale worker=1
```
