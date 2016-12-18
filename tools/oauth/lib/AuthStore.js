'use strict';

var nano = require('nano');
var LRU = require("lru-cache");
var Boom = require('boom');

var cache = new LRU(10000);

const url = config => `${config.dbProtocol}://${config.dbUsername}:${config.dbPassword}@${config.dbHost}`
const urlWithDb = config => `${url(config)}/${config.dbAuths}`
const setup = config => nano(url(config))
const setupWithDb = config => nano(urlWithDb(config));

function AuthStore(config, logger, statsdClient) {
    var cloudant = setupWithDb(config)

    this.insertAuthDocument = function insertAuthDocument(tid, doc) {
        var method = 'insertAuthDocument';
        statsdClient.increment('cloudant.' + method + '.count');
        logger.info(tid, method, 'inserting document');

        return new Promise(function(resolve, reject) {
            statsdClient.increment('cloudant.request.count');
            var start = new Date().getTime();
            cloudant.insert(doc, function(err, body) {
                var duration = new Date().getTime() - start;
                statsdClient.timing('cloudant.request.time', duration);
                statsdClient.timing('cloudant.' + method + '.time', duration);
                if(!err) {
                    logger.info(tid, method, 'inserted document for', body._id);
                    resolve(body);
                }
                else {
                    var error = Boom.wrap(err, err.statusCode);
                    statsdClient.increment('cloudant.request.error.count');
                    logger.error(tid, method, 'error while inserting document', error.statusCode, error.message, urlWithDb(config));
                    reject(error);
                }
            });
        });
    };

    // TODO: reject error cases
    this.getAuthDocumentForSubject = function getAuthDocumentForSubject(tid, subject, useCache = true) {
        var method = 'getAuthDocumentForSubject';
        statsdClient.increment('cloudant.' + method + '.count');

        logger.info(tid, method, 'getting auth document for', subject);
        return new Promise(function(resolve, reject) {
            var cached = cache.get(subject);
            if(cached && useCache) {
                statsdClient.increment('cloudant.' + method + '.cache.hit.count');
                logger.info(tid, method, 'found auth document in cache for', subject);
                resolve(cached);
            }
            else {
                statsdClient.increment('cloudant.' + method + '.cache.miss.count');
                statsdClient.increment('cloudant.request.count');
                var start = new Date().getTime();
                cloudant.get(subject, function(err, body) {
                    var duration = new Date().getTime() - start;
                    statsdClient.timing('cloudant.request.time', duration);
                    statsdClient.timing('cloudant.' + method + '.time', duration);
                    if(!err) {
                        logger.info(tid, method, 'found auth document in database for', subject);
                        cache.set(subject, body);
                        resolve(body);
                    }
                    else {
                        var error = Boom.wrap(err, err.statusCode);
                        statsdClient.increment('cloudant.request.error.count');
                        logger.error(tid, method, 'document for', subject, 'not found', error.statusCode, error.message);
                        reject(error);
                    }
                });
            }
        });
    };

    this.getAuthDocumentForUuidKey = function getAuthDocumentForUuidKey(tid, uuid, key) {
        var method = 'getAuthDocumentForUuid';
        statsdClient.increment('cloudant.' + method + '.count');

        logger.info(tid, method, 'getting auth document for', uuid);
        return new Promise(function(resolve, reject) {
            statsdClient.increment('cloudant.request.count');
            var start = new Date().getTime();
            cloudant.view('subjects', 'identities', {key: [uuid, key]}, function(err, result) {
                var duration = new Date().getTime() - start;
                statsdClient.timing('cloudant.request.time', duration);
                statsdClient.timing('cloudant.' + method + '.time', duration);
                if(!err) {
                    if(result.rows && result.rows.length === 1) {
                        logger.info(tid, method, 'found auth document for', uuid);
                        resolve(result.rows[0]);
                    }
                    else {
                        logger.error(tid, method, 'auth document for', uuid, 'not found');
                        statsdClient.increment('cloudant.request.error.count');
                        reject(Boom.create(404, 'auth document for' + uuid + 'not found'));
                    }
                }
                else {
                    var error = Boom.wrap(err, err.statusCode);
                    logger.error(tid, method, error.statusCode, error.message);
                    reject(error);
                }
            });
        });
    };

    this.updateAuthDocument = function updateAuthDocument(tid, doc) {
        var method = 'updateAuthDocument';
        statsdClient.increment('cloudant.' + method + '.count');

        cache.del(doc._id);
        logger.info(tid, method, 'updating auth document with id', doc._id);
        return new Promise(function(resolve, reject) {
            statsdClient.increment('cloudant.request.count');
            var start = new Date().getTime();
            cloudant.insert(doc, doc._id, function(err, body) {
                cache.del(doc._id);
                var duration = new Date().getTime() - start;
                statsdClient.timing('cloudant.request.time', duration);
                statsdClient.timing('cloudant.' + method + '.time', duration);
                if(!err) {
                    logger.info(tid, method, 'updated auth document with id', body._id);
                    resolve(body);
                }
                else {
                    var error = Boom.wrap(err, err.statusCode);
                    statsdClient.increment('cloudant.request.error.count');
                    logger.error(tid, method, 'error while updating document', error.statusCode, error.message);
                    reject(error);
                }
            });
        });
    };

    this.deleteAuthDocument = function deleteAuthDocument(tid, subject) {
        var method = 'deleteAuthDocument';
        statsdClient.increment('cloudant.' + method + '.count');

        cache.del(subject);
        logger.info(tid, method, 'deleting document of', subject);
        return this.getAuthDocumentForSubject(tid, subject).then(function(doc) {
            cache.del(subject);
            return new Promise(function(resolve, reject) {
                statsdClient.increment('cloudant.request.count');
                var start = new Date().getTime();
                cloudant.destroy(doc._id, doc._rev, function(err, body) {
                    cache.del(subject);
                    var duration = new Date().getTime() - start;
                    statsdClient.timing('cloudant.request.time', duration);
                    statsdClient.timing('cloudant.' + method + '.time', duration);
                    if(!err) {
                        logger.info(tid, method, 'deleted document of', subject);
                        resolve(body);
                    }
                    else {
                        statsdClient.increment('cloudant.request.error.count');
                        var error = Boom.wrap(err, err.statusCode);
                        logger.error(tid, method, 'error while deleting document', error.statusCode, error.message);
                        reject(error);
                    }
                });
            });
        });
    };
}

module.exports = AuthStore;
