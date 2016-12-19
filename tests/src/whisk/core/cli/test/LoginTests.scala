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

package whisk.core.cli.test

import java.net.ServerSocket

import scala.concurrent.Await
import scala.concurrent.Future
import scala.concurrent.TimeoutException
import scala.concurrent.duration.DurationInt
import scala.io.BufferedSource

import org.junit.runner.RunWith
import org.openqa.selenium.By
import org.openqa.selenium.WebElement
import org.openqa.selenium.firefox.FirefoxDriver
import org.openqa.selenium.firefox.FirefoxProfile
import org.openqa.selenium.support.ui.ExpectedConditions
import org.openqa.selenium.support.ui.WebDriverWait
import org.scalatest.Finders
import org.scalatest.junit.JUnitRunner
import org.scalatest.selenium.WebBrowser

import common.TestHelpers
import common.TestUtils.ERROR_EXIT
import common.TestUtils.SUCCESS_EXIT
import common.Wsk
import common.WskActorSystem
import common.WskAdmin
import common.WskProps
import common.WskTestHelpers

import scala.util.Properties

/**
 * Tests for testing the CLI "login" subcommand.  These tests require a deployed backend.
 */
@RunWith(classOf[JUnitRunner])
class LoginTests
    extends TestHelpers
    with WebBrowser
    //with ScalaFutures
    with WskActorSystem
    with WskTestHelpers {

    implicit val wskprops = WskProps()
    val wsk = new Wsk
    //val (cliuser, clinamespace) = WskAdmin.getUser(wskprops.authKey)

    behavior of "Wsk login"

    val profileDir = java.nio.file.Files.createTempDirectory("openwhisk")

    //
    // this is where we initialize the selenium driver
    //
    val firefoxProfile = new FirefoxProfile(profileDir.toFile())
    firefoxProfile.setPreference("xpinstall.signatures.required", false)
    firefoxProfile.setAcceptUntrustedCertificates(true);
    implicit val driver = new FirefoxDriver(firefoxProfile);
    driver.manage.deleteAllCookies

    it should "reset github authorization, so we can re-test granting authorization" in {
        go to "https://github.com/settings/applications"

        doGitHubLogin

        try {
            parent(parent(parent(driver.findElementByLinkText("OpenWhisk CLI auth test"))))
                .findElement(By.linkText("Revoke"))
                .click()

            new WebDriverWait(driver, 10)
                .until(ExpectedConditions.presenceOfElementLocated(By.className("js-revoke-access-form")))
                .findElement(By.tagName("button"))
                .click()
        } catch {
            // failure here is OK; it means our test user hasn't ever granted access to this oauth app
            case _: Throwable => true
        }

        System.out.println("Good")
    }

    it should "reject a bogus provider" in {
        val (_, rr) = doWskLogin(provider = "randomgarbage", expectedExitCode = ERROR_EXIT)
        rr.stderr should include("Unsupported oauth provider")
    }

    it should "succeed in logging in to github" in {
        if (Properties.envOrNone("GITHUB_USER").isDefined && Properties.envOrNone("GITHUB_PASSWORD").isDefined) {
            val (future, rr) = doWskLogin(provider = "github", authorizer = doAuthorizeGitHub)

            Await.ready(future, 20 seconds).value.get match {
                case _ =>
                    // System.out.println(rr.stdout)
                    // System.err.println(rr.stderr)

                    // the backend should be happy
                    rr.stdout should include("ok")
            }
        }
    }

    /*it should "succeed in logging in to google" in {
        val rr = wsk.login.login(provider = "google")
        implicit val driver = attach

        // log in to google
        pageTitle should include ("Sign in")
        textField("Email").value = "email address..."
        click on id("next")
        textField("Passwd").value = ""
        click on name("commit")
        
        // authorize!
        pageTitle should include ("Authorize OpenWhisk")
        click on name("authorize")

        rr.stdout should include("ok")
    }*/

    //def profile = profileDir.toString()

    /**
     * @return the parent WebElement of a given WebEleemnt
     */
    def parent(e: WebElement) = e.findElement(By.xpath("./.."));

    /**
     * Talk to the CLI to initialize a `wsk login`
     */
    def doWskLogin(provider: String, expectedExitCode: Int = SUCCESS_EXIT, authorizer: Authorizer = doAuthorizeNoop) = {
        val (port, future) = listen(authorizer)

        (future, wsk.login.login(
            provider = provider,
            expectedExitCode = expectedExitCode,
            port = port))
    }

    type Authorizer = () => Unit

    def doAuthorizeNoop() = {}

    /**
     * Talk to the web browser to initialize a github login
     */
    def doGitHubLogin = {
        // log in to github
        new WebDriverWait(driver, 30)
            .until(ExpectedConditions.titleContains("Sign in"))
        textField("login_field").value = sys.env("GITHUB_USER")
        pwdField("password").value = sys.env("GITHUB_PASSWORD")
        click on name("commit")
    }

    def doAuthorizeGitHub() = {
        //doGitHubLogin this is now done above, in revoking the access

        // authorize!
        new WebDriverWait(driver, 10)
            .until(ExpectedConditions.titleContains("Authorize OpenWhisk"))
        click on name("authorize")
    }

    /**
     * @return (port we're listening on, future value of completion)
     */
    def listen(authorizer: Authorizer): (Integer, Future[Boolean]) = {
        val server = new ServerSocket(0)

        val future = Future[Boolean] {
            try {
                val s = server.accept()
                val in = new BufferedSource(s.getInputStream()).getLines()

                val url = in.next();
                go to url
                authorizer()
                //val out = new PrintStream(s.getOutputStream())

                //out.println(in.next())
                //out.flush()
                s.close()
            } finally {
                server.close()
            }

            //System.out.println("ALL DONE WITH LISTEN")
            true
        }

        // System.out.println(s"LISTENING on ${server.getLocalPort}")
        (server.getLocalPort, future)
    }
}