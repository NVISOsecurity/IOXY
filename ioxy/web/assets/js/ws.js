//  Variables
let socket = new WebSocket("ws://" + window.location.host + "/ws");
socket.onopen = function (e) {
    appendInConsole("Connection with MOXY established");
};

var way = "";

function isJson(str) {
    try {
        JSON.parse(str);
    } catch (e) {
        return false;
    }
    return true;
}

function sendWsPayload(payload) {
    socket.send(payload);
}

socket.onmessage = function (event) {
    message = event.data;
    if (isJson(message)) {
        message = JSON.parse(message);
        if (message.Intercept) {
            $("#interceptCol").removeClass("disableDiv");
            $('input[type=checkbox][data-toggle^=toggle]').bootstrapToggle("disable");
            appendInConsole("Incoming message to forward | Client " + message.Way +" Distant Broker | Topic : " + "\"" + message.Topic + "\" | Payload : " + "\"" + atob(message.Payload) + "\"");
            $("#publishTopic").attr("placeholder", message.Topic);
            $("#publishTopic").val(message.Topic);
            $("#payload").attr("placeholder", atob(message.Payload));
            $("#payload").val(atob(message.Payload));
            way = message.Way;
            if (message.Way === ">"){
                $("#way1").addClass("waveLTR");
                $("#way2").removeClass("waveRTL");
            } else if(message.Way === "<"){
                $("#way2").addClass("waveRTL");
                $("#way1").removeClass("waveLTR");
            }
        }
    } else {
        var way;
        if (message.includes('client > broker')) {
            way = ">";
        } else if (message.includes('> -')){
            way = ">";
        } else if (message.includes("client < broker")){
            way = "<";
        } else if (message.includes('< -')){
            way = "<";
        } else {
            way = "";
        }
        if (way != "") {
            animateConnections(way,2);
        }
        appendInConsole(message);
    }
};

socket.onclose = function (event) {
    if (event.wasClean) {
        appendInConsole(`[close] Connection with MOXY closed cleanly, code=${event.code} reason=${event.reason}`);
    } else {
        appendInConsole('[close] Connection with MOXY died');
    }
};

socket.onerror = function (error) {
    appendInConsole(`[error] ${error.message}`);
};


function forwardMessage() {
    $("#way1").removeClass("waveRTL");
    $("#way1").removeClass("waveLTR");
    $("#way2").removeClass("waveRTL");
    $("#way2").removeClass("waveLTR");
    animateConnections(way,2);
    message = {"topic" : $("#publishTopic").val(),"payload" : $("#payload").val()};
    socket.send(JSON.stringify(message));
    appendInConsole("Forwarding message | Topic : " + "\"" + message.topic + "\" | Payload : " + "\"" + message.payload + "\"");
    $("#publishTopic").val("");
    $("#publishTopic").attr("placeholder", "Waiting for incoming message");
    $("#payload").val("");
    $("#payload").attr("placeholder", "Waiting for incoming message");
    $("#interceptCol").addClass("disableDiv");
    $('input[type=checkbox][data-toggle^=toggle]').bootstrapToggle("enable");
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function animateConnections(way, passes = 1,iter = 1) {
    var obj = "";
    var className = "";
    if (way === ">") {
        className = "waveLTR";
        obj = [$("#way1"),$("#way2")];
    } else if (way === "<") {
        className = "waveRTL";
        obj = [$("#way2"),$("#way1")];
    } else {
        return;
    }
    for (i = 0; i < iter; i++) {

        obj[0].addClass(className);
        await sleep(500);
        obj[0].removeClass(className);
        if (passes > 1) {
            obj[1].addClass(className);
            await sleep(500);
            obj[1].removeClass(className);
        }
    }
}