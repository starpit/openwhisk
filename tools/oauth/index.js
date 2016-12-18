const express = require('express');
const request = require('request-promise');
const HttpStatus = require('http-status-codes');
var StatsD = require('node-statsd');
const config = require('./lib/config');
const logger = require('./lib/Logger');

const app = express();
const bodyParser = require('body-parser');
app.use(bodyParser.urlencoded({ extended: false }));
app.use(bodyParser.json());

// Swagger docs
app.use('/docs', express.static(__dirname + '/docs'));

// RAS endpoint
app.get("/ping", function pong(req, res) {
    res.send({msg: 'pong'});
});

logger.info('#tid_oauth_0', 'startup', 'configuration loaded from environment', config);

function startService() {

    var statsdClient = new StatsD({
        'host': config.statsdHost,
        'port': config.statsdPort,
        'prefix': 'openwhisk.oauth.'
    });

    const middleware = {};
    const rights = {};
    
    const adminRouter = express.Router();
    const v1Router = express.Router();
    require('./apis/v1/Entitlement').init(app, v1Router, rights, middleware, config, logger, statsdClient);
    require('./apis/v1/Users').init(app, v1Router, middleware, config, logger, statsdClient);
    //require('./apis/v1/Monitoring').init(app, v1Router, middleware, config, logger, statsdClient);
    app.use('/oauth/v1', v1Router);

    app.use('/admin', adminRouter);

    logger.info('#tid_oauth_0', 'startService', 'server started, listening on', config.servicePort);
    console.log(config.servicePort)
    app.listen(config.servicePort);
}
startService();
