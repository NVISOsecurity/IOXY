let console = {};
            
// Getting div to insert logs
let logger = document.getElementById("logger");

function containDate(text){
  var reg = new RegExp('\\[\\d{4}-\\d{2}-\\d{2}\\s\\d{2}:\\d{2}:\\d{2}\\.\\d{2}\\]');
  if (text.match(reg)){
    return true;
  } else {
    return false;
  }
}

function getFormattedTime() {
  var d = new Date();
  var n = d.toISOString();
  n = n.replace("T"," ").replace("Z","");
  return "["+n.substring(0, n.length - 1)+"]";
}

function appendInConsole(text) {
    let element = document.createElement("div");
    let time = containDate(text) ? "" : getFormattedTime().toString()+" ";
    let txt = document.createTextNode(time+text);
    element.classList.add("py-1");
    element.appendChild(txt);
    logger.appendChild(element);
    logger.scrollTop = logger.scrollHeight;
    return;
}

//Clear Console
async function clearConsole() {
    logger.innerHTML = "";
    appendInConsole("Console cleared.")
    logger.innerHTML += "<button class='btn btn-primary' type='button' onclick='clearConsole()' style='position : absolute;bottom : 1vh;right:3vh'>Clear console</button>";
    await new Promise(r => setTimeout(r, 2000));
    logger.removeChild(logger.childNodes[0]);
    return;
}