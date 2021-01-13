window.onload = function(event) {
  var div = document.createElement('div');
  div.innerText = "Hello from Javascript!";

  var body = document.querySelector('body');
  body.appendChild(div);
};
