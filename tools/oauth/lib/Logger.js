var _ = require('lodash');
var moment = require('moment');
var winston = require('winston');

var logger = new winston.Logger({
    transports: [
        new winston.transports.Console({
            timestamp: function() {
                return moment.utc().format("YYYY-MM-DDTHH:mm:ss.SSS") + 'Z';
            },
            formatter: function(options) {
                // Return string will be passed to logger.
                return '[' + options.timestamp() +'] ['+ options.level.toUpperCase() +'] '+  options.message;
            }
        })
    ]
});

/**
 * Transforms object arguments into readable strings
 * @param  {List} args List of arguments
 * @return {String} string of all arguments, delimited by a space
 */
const message = args => args.map(arg => _.isObject(arg) ? JSON.stringify(arg) : arg).join(' ');

// FORMAT: s"[$time] [$category] [$id] [$componentName] [$name] $message"
module.exports = {
    info: function(tid, name, ...args) {
        logger.info('['+tid+']', '['+name+']', message(args));
    },
    warn: function(tid, name, ...args) {
        logger.warn('['+tid+']', '['+name+']', message(args));
    },
    error: function(tid, name, ...args) {
        logger.error('['+tid+']', '['+name+']', message(args));
    },
    debug: function(tid, name, ...args) {
        logger.debug('['+tid+']', '['+name+']', message(args));
    }
};
