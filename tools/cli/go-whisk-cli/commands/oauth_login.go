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
	"net"
	"sync"
	"bytes"
	"errors"
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
const redirectPort = 15231

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
				fmt.Fprintln(os.Stderr, "Error killing browser", kill_err)
			}
		}

		browserCmd = nil
	}
}

// something goofed up along the way
func allBad(res http.ResponseWriter, req *http.Request) {
	killBrowser()
	fmt.Println("Exiting 1")
	os.Exit(1)
}

// we got our subject key
func finishUp(body []byte, res http.ResponseWriter, req *http.Request) error {
	killBrowser()

	if _, ok := os.LookupEnv("PORT_MODE"); ok {
		// in PORT_MODE, we won't be updating the properties file
		return nil
	}
	
	// now we need to write out the properties
	
        if props, read_err := marshalling.ReadProps(Properties.PropsFile); read_err == nil {
		var subject SubjectType

		if parse_err := json.Unmarshal(body, &subject); parse_err == nil {
			var auth = fmt.Sprintf("%s:%s", subject.Uuid, subject.Key)
			props["AUTH"] = auth
			props["NAMESPACE"] = "_"
		
			if write_err := marshalling.WriteProps(Properties.PropsFile, props); write_err == nil {
				// hurray, we're all done
				return nil

			} else {
				return write_err
			}
		} else {
			return parse_err
		}
	} else {
		return read_err
	}
}

func onCodeCallback(providerName string, wg sync.WaitGroup) func (res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		fmt.Println("Code callback")
		
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
				fmt.Fprintln(os.Stderr, err);
				allBad(res, req);
				return;
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				fmt.Fprintln(os.Stderr, "Error communicating with backend. Got statusCode",
					resp.StatusCode);
				allBad(res, req)
			} else {
				//
				// we should now have a subject document
				//
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error reading response", err)
					allBad(res, req)

				} else {
					finishup_err := finishUp(body, res, req)
					if (finishup_err == nil) {
						fmt.Println("ok")
						os.Exit(0)
						//wg.Done()

					} else {
						fmt.Fprintln(os.Stderr, finishup_err)
						allBad(res, req)
					}
				}
			}
		} else {
			//
			// then we're just serving up a static file
			//
			fmt.Println("Sending %v", req.URL.Path)
			sendFile(req.URL.Path, res, req)
			wg.Done()
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
		if port, ok := os.LookupEnv("PORT_MODE"); ok {
			fmt.Println("PORT MODE. Using port", port)
			//cmd = "firefox"
			//args = []string{"--profile", profile, "--new-tab"}
			//fmt.Println("URL", Url)
			conn, err := net.Dial("tcp", "localhost:" + port)
			fmt.Fprintf(conn, "%s", Url)
			conn.Close()
			return nil, err
		} else {
			cmd = "xdg-open"
		}
	}
	args = append(args, Url)
	command := exec.Command(cmd, args...)
	// fmt.Println("COMMAND",command)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return command, command.Start()
}

/*func waitForBrowserToExit() {
	if browserCmd != nil {
		browserCmd.Wait()

		if browserCmd != nil {
			os.Exit(1)
		}
	}
}*/

func doLoginWithProvider(providerName string, provider ProviderType) error {
	//
	// oauth requires that we use a browser to accept the user's
	// credentials, and that we service a redirect_uri that the identity
	// provider will call with the oauth code (this code is a partial
	// assurance that the user is who they claim to be; the rest of the
	// oauth handshake must be handled on the backend, in order to avoid
	// exposing any of our oauth application secrets to the client)
	//
	var wg sync.WaitGroup
	wg.Add(1)
	
	http.HandleFunc("/", onCodeCallback(providerName, wg));
	go http.ListenAndServe(fmt.Sprintf(":%d", redirectPort), nil);

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
		return err
	}

	redirect_uri := fmt.Sprintf("http://localhost:%d", redirectPort)
	fmt.Println("REDIRECT_URI",redirect_uri)

	parameters := url.Values{}
	parameters.Add("client_id", provider.Credentials.Client_id)
	parameters.Add("redirect_uri", redirect_uri)
	for k,v := range provider.Authorization_endpoint_query {
		parameters.Add(k,v)
	}
	Url.RawQuery = parameters.Encode()

	if cmd, err := openBrowser(Url.String()); err != nil {
		fmt.Fprintln(os.Stderr, "Error opening browser", err)
		return err
	} else {
		fmt.Println("Good, browser opened")
		browserCmd = cmd
		// defer waitForBrowserToExit()

		wg.Wait()
		fmt.Println("All done")
		// if we get here, we have success
		os.Exit(0)
		return nil
	}
} /* end of doLoginWithProvider */

func doLoginWithProviderName(providerName string) error {
	file, err := config.Load("oauth-providers.json")//ioutil.ReadFile("conf/oauth/providers.json")
	if err != nil {
		return err
	}

	var providers ProvidersType
	if parse_err := json.Unmarshal(file, &providers); parse_err != nil {
		return parse_err
	}
	
	if provider, ok := providers[providerName]; ok {
		return doLoginWithProvider(providerName, provider)

	} else {
		return errors.New(fmt.Sprintf("Unsupported oauth provider: %s", providerName))
	}

} /* end of doLoginWithProviderName */

/*func main() {
	var providerName = os.Args[1]
	fmt.Println("Using this provider: %s", providerName)
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
	  return doLoginWithProviderName(providerName)
  },
}
