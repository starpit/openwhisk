const url = require('url'),
      path = require('path'),
      fs = require('fs'),
      uuid = require('uuid'),
      randomstring = require('randomstring'),
      request = require('request'),
      getTransactionId = require('../../lib/utils').getTransactionId;

// config information about the providers
// todo fetch the google three fields from https://accounts.google.com/.well-known/openid-configuration
const providers = JSON.parse(fs.readFileSync(path.join(__dirname, path.join('..', '..', 'conf', 'providers.json')), 'utf8'));

/**
 * Send an auth key back to the user
 *
 */
function sendKey(status, doc, res) {
    delete doc["_id"]
    delete doc["_rev"]
    const stringified = JSON.stringify(doc);
    
    res.writeHead(status, {
	'Content-Length': stringified.length,
	'Content-Type': 'application/json'
    });

    res.write(stringified);
    res.end();
}
const allGood = sendKey.bind(undefined, 200);
const allBad = sendKey.bind(undefined, 500);

const createAuthDocument = subject => ({
    _id: subject,
    subject: subject,
    key: randomstring.generate(64),
    uuid: uuid.v4()
})

/**
 * Listen for requests for oauth logins from the clients
 *
 */
const doOAuthLogin = (logger, authstore, req, res) => {
    //
    // the client is giving has an oauth code; we want to exchange
    // this for an access_token, and, from there, for some identifying
    // information from the user's profile. first thing's first...
    //
    const tid = getTransactionId(req)
    const method = 'authenticate'
    logger.info(tid, method, 'oauth authenticate initiated')
    
    //var url_parts = url.parse(req.url, true);
    //var query = url_parts.query;

    // here is the oauth code the user gave us
    var code = req.body.code;//query.code;
    
    var providerName = req.body.provider;//query.provider;
    var provider = providers[providerName];

    //
    // this is the body of our access_token request
    //
    var form = {
	client_id: provider.credentials.client_id,
	client_secret: provider.credentials.client_secret,
	code: code
    };
    if (provider.token_endpoint_form) {
	for (var x in provider.token_endpoint_form) {
	    form[x] = provider.token_endpoint_form[x];
	}
    }

    //
    // form the request options for the access_token
    //
    var options = {
	url: provider.token_endpoint,
	method: 'POST',
	headers: {
	    'Content-Type': 'application/json'
	},
    };
    if (provider.token_endpoint_form_is_json) {
	options.headers['Accept'] = 'application/json';
	options.body = form;
	options.json = true;
    } else {
	options.form = form;
    }

    //
    // ok, here we go, exchanging the oauth code for an access_token
    //
    request(options, (err, response, body) => {
	if (err) {
	    console.log(JSON.stringify(err));
	    logger.error(tid, method, 'exchanging code for access_token')
	    allBad(undefined, res);
	} else {
	    //
	    // all right, we now have an access_token
	    //
	    if (typeof body == "string") {
		body = JSON.parse(body);
	    }
	    
	    //
	    // now we request the user's profile, so that we have some
	    // persistent identifying information; e.g. email address
	    // for account handle
	    //
	    request({
		url: provider.userinfo_endpoint,
		method: 'GET',
		headers: {
		    'Accept': 'application/json',
		    'Authorization': (provider.authorization_type || 'token') + ' ' + body.access_token,
		    "User-Agent": "OpenWhisk"
		}
	    }, (err2, response2, body2) => {
		if (err2) {
		    logger.error(tid, method, 'exchanging access_token for profile')
		    allBad(undefined, res);
		} else {
		    //
		    // great, now we have the profile!
		    //
		    if (typeof body2 == "string") {
			body2 = JSON.parse(body2);
		    }

		    const subject = body2[provider.userinfo_identifier]
		    logger.info(tid, method, 'user', subject, 'oauth authenticate complete')

		    authstore.getAuthDocumentForSubject(tid, subject, false)
			.then(doc => {
			    logger.info(tid, method, 'user', subject, 'already existed, sending document');
			    allGood(doc, res)
			})
			.catch(err => {
			    if(err.output.statusCode === 404) {
				logger.info(tid, method, 'user', subject, 'does not exist, creating a new one');
				
				const doc = createAuthDocument(subject)
				
				authstore.insertAuthDocument(tid, doc)
				    .then(() => allGood(doc, res))
				    .catch(err => res.status(err.output.statusCode).json({error: err.message}))
			    } else {
				res.status(err.output.statusCode).json({error: err.message})
			    }
			});
		}
	    });
	}
    });
};
	  
module.exports.init = function(app, router, middleware, config, logger, statsdClient) {
    const AuthStore = require('../../lib/AuthStore')
    const authstore = new AuthStore(config, logger, statsdClient)
    
    router.post('/authenticate', doOAuthLogin.bind(undefined, logger, authstore))
}
