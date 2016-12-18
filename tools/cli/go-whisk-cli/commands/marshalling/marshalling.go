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

package marshalling

import (
    "os"
    "fmt"
    "bufio"
    "errors"
    "strings"

    "../../../go-whisk/whisk"
    "../../wski18n"
)

func ReadProps(path string) (map[string]string, error) {

    props := map[string]string{}

    file, err := os.Open(path)
    if err != nil {
        // If file does not exist, just return props
        whisk.Debug(whisk.DbgWarn, "Unable to read whisk properties file '%s' (file open error: %s); falling back to default properties\n" ,path, err)
        return props, nil
    }
    defer file.Close()

    lines := []string{}
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }

    props = map[string]string{}
    for _, line := range lines {
        kv := strings.Split(line, "=")
        if len(kv) != 2 {
            // Invalid format; skip
            continue
        }
        props[kv[0]] = kv[1]
    }

    return props, nil

}

func WriteProps(path string, props map[string]string) error {

    file, err := os.Create(path)
    if err != nil {
        whisk.Debug(whisk.DbgError, "os.Create(%s) failed: %s\n", path, err)
        errStr := fmt.Sprintf(
            wski18n.T("Whisk properties file write failure: {{.err}}",
                map[string]interface{}{"err": err}))
        werr := whisk.MakeWskError(errors.New(errStr), whisk.EXITCODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
        return werr
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    defer writer.Flush()
    for key, value := range props {
        line := fmt.Sprintf("%s=%s", strings.ToUpper(key), value)
        _, err = fmt.Fprintln(writer, line)
        if err != nil {
            whisk.Debug(whisk.DbgError, "fmt.Fprintln() write to '%s' failed: %s\n", path, err)
            errStr := fmt.Sprintf(
                wski18n.T("Whisk properties file write failure: {{.err}}",
                    map[string]interface{}{"err": err}))
            werr := whisk.MakeWskError(errors.New(errStr), whisk.EXITCODE_ERR_GENERAL, whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return werr
        }
    }
    return nil
}
