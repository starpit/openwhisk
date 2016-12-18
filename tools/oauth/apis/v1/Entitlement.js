module.exports.init = function(app, router, rights, middleware, config, logger, statsdClient) {

    app.get('/namespaces', function(req, res) {
        res.status(HttpStatus.NOT_IMPLEMENTED).json({error: 'Explicit checks are not implemented'});
    });

    app.get('/check', function(req, res) {
        res.status(HttpStatus.NOT_IMPLEMENTED).json({error: 'Explicit checks are not implemented'});
    });

    app.post('/grant', function(req, res) {
        res.status(HttpStatus.NOT_IMPLEMENTED).json({error: 'Explicit checks are not implemented'});
    });

    app.post('/revoke', function(req, res) {
        res.status(HttpStatus.NOT_IMPLEMENTED).json({error: 'Explicit checks are not implemented'});
    });
};
