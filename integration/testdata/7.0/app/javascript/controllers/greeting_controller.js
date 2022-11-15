import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  connect() {
    var div = document.createElement('div');
    div.innerText = 'Hello from Javascript!';

    this.element.appendChild(div);
  }
}
