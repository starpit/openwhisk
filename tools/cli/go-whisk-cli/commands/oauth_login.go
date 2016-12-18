/*
 * Copyright 2015-2016 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"os"
	"fmt"
	"bytes"
	"runtime"
	"syscall"
	"io/ioutil"
	"os/exec"
	"net/url"
	"net/http"
	"encoding/json"

	"../wski18n"
	"../config"
	"./marshalling"
	
	"github.com/spf13/cobra"
)

// change this to whatever we decide to use for the controller route
const backendURI = "http://localhost:10014/oauth/v1/authenticate";

// DO NOT CHANGE THIS, without also changing the oauth application registrations
const redirectPort = 15231;

// DANGER GLOBAL: this is the *exec.Cmd of the browser subprocess
var browserCmd *exec.Cmd

// datatype hierarchy for the oauth-providers.json config
type ProvidersType map[string]ProviderType
type ProviderType struct {
	Authorization_endpoint string
	Authorization_endpoint_query map[string]string
	Credentials CredentialsType
}
type CredentialsType struct {
	Client_id string
}

// datatype hierarchy for the subject document
type SubjectType struct {
	Subject string
	Key string
	Uuid string
}

func sendFile(f string, res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, f)
}

func killBrowser() {
	if browserCmd != nil {
		pgid, err := syscall.Getpgid(browserCmd.Process.Pid)
		if err == nil {
			kill_err := syscall.Kill(-pgid, 15)  // note the minus sign

			if kill_err != nil {
				fmt.Printf("Error killing browser %s\n", kill_err)
			}
		}

		browserCmd = nil
	}
}

// something goofed up along the way
func allBad(res http.ResponseWriter, req *http.Request) {
	killBrowser()
	os.Exit(1)
}

// se got our subject key
func allGood(body []byte, res http.ResponseWriter, req *http.Request) {
	killBrowser()

	// now we need to write out the properties
	
        if props, read_err := marshalling.ReadProps(Properties.PropsFile); read_err == nil {
		var subject SubjectType

		if parse_err := json.Unmarshal(body, &subject); parse_err == nil {
			var auth = fmt.Sprintf("%s:%s", subject.Uuid, subject.Key)
			props["AUTH"] = auth
			props["NAMESPACE"] = "_"
		
			if write_err := marshalling.WriteProps(Properties.PropsFile, props); write_err == nil {
				// hurray, we're all done
				fmt.Printf("ok\n")
				os.Exit(0)
			} else {
				fmt.Printf("Error writing properties %v\n", write_err)
			}
		} else {
			fmt.Printf("Error parsing json %v\n", parse_err)
		}
	} else {
		fmt.Printf("Error reading properties %v\n", read_err)
	}

	os.Exit(1)
}

func onCodeCallback(providerName string) func (res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		//
		// cool, we should now have an oauth code
		//
		var query = req.URL.Query()

		if codes, ok := query["code"]; ok {
			killBrowser()
			
			//
			// pass this code to the backend, and get back an auth_key
			//
			var code = codes[0]
			var jsonStr = fmt.Sprintf("{\"code\": \"%s\", \"provider\": \"%s\"}", code, providerName)
			var jsonBytes = []byte(jsonStr)
			req, err := http.NewRequest("POST", backendURI, bytes.NewBuffer(jsonBytes))
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("All bad %s\n", err);
				allBad(res, req);
				return;
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				fmt.Printf("All bad %s\n", resp.Status)
				allBad(res, req)
			} else {
				//
				// we should now have a subject document
				//
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("Error reading response %s\n", err)
					allBad(res, req)

				} else {
					// fmt.Printf("Login successful %s\n", body)
					allGood(body, res, req)
				}
			}
		} else {
			//
			// then we're just serving up a static file
			//
			fmt.Printf("Sending %v", req.URL.Path)
			sendFile(req.URL.Path, res, req)
		}
	}
}

// open opens the specified URL in the default browser of the user.
func openBrowser(Url string) (*exec.Cmd, error) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, Url)
	command := exec.Command(cmd, args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return command, command.Start()
}

func doLoginWithProvider(providerName string, provider ProviderType) bool {
	//
	// oauth requires that we use a browser to accept the user's
	// credentials, and that we service a redirect_uri that the identity
	// provider will call with the oauth code (this code is a partial
	// assurance that the user is who they claim to be; the rest of the
	// oauth handshake must be handled on the backend, in order to avoid
	// exposing any of our oauth application secrets to the client)
	//
	http.HandleFunc("/", onCodeCallback(providerName));
	defer http.ListenAndServe(fmt.Sprintf(":%d", redirectPort), nil);

	//
	// when the server is up, we are ready to open up a browser so
	// that the user can start the login process
	//
	// what happens here: we open the browser to the provider's
	// authorization endpoint, specifying a redirect_uri that points
	// back to the server we just started up; the provider will call
	// us back with the oauth code
	//
	var Url *url.URL
	Url, err := url.Parse(provider.Authorization_endpoint)
	if err != nil {
		return false
	}

	parameters := url.Values{}
	parameters.Add("client_id", provider.Credentials.Client_id)
	parameters.Add("redirect_uri", fmt.Sprintf("http://localhost:%d", redirectPort))
	for k,v := range provider.Authorization_endpoint_query {
		parameters.Add(k,v)
	}
	Url.RawQuery = parameters.Encode()

	if cmd, err := openBrowser(Url.String()); err != nil {
		fmt.Printf("Error opening browser %s\n", err)
		return false
	} else {
		browserCmd = cmd
		return true
	}
} /* end of doLoginWithProvider */

func doLoginWithProviderName(providerName string) {
	file, e := config.Load("oauth-providers.json")//ioutil.ReadFile("conf/oauth/providers.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		//os.Exit(1)
		return
	}
	var providers ProvidersType
	json.Unmarshal(file, &providers)

	// fmt.Print(providers)
	
	if provider, ok := providers[providerName]; ok {
		doLoginWithProvider(providerName, provider)

	} else {
		fmt.Printf("Unsupported auth provider %s\n", providerName);
	}

}

/*func main() {
	var providerName = os.Args[1]
	fmt.Printf("Using this provider: %s\n", providerName)
	doLoginWithProviderName(providerName)
}*/


var oauth_loginCmd = &cobra.Command{
  Use:   "login PROVIDER",
  Short: wski18n.T("login with a given provider"),
  PreRunE:       setupClientConfig,
  RunE: func(cmd *cobra.Command, args []string) error {
      if whiskErr := checkArgs(args, 1, 1, "login",
	      wski18n.T("Please specify an identity provider.")); whiskErr != nil {
		      return whiskErr
	      }

	  var providerName = args[0]
	  doLoginWithProviderName(providerName)

	  return nil
  },
}
