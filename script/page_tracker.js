"use strict";

var fs = require("fs");
var system = require("system");
var args = system.args;
var url = args[1];

if (url == "") {
    system.stderr.writeLine("ERROR: Missing required page URL argument");
    phantom.exit(1)
}

var webpage = require("webpage");
var page = webpage.create();
page.open(url, function(status) {
    if (status !== "success") {
        system.stderr.writeLine("ERROR: Fail to load page: " + url);
        phantom.exit(1)
    }
});

page.onLoadFinished = function () {
    system.stdout.write(page.content);
    phantom.exit();
};



