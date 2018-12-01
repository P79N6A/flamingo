function getCookie(cname){
  var name = cname + "=";
  var ca = document.cookie.split(';');
  for(var i=0; i<ca.length; i++) {
    var c = ca[i].trim();
    if (c.indexOf(name)==0) return c.substring(name.length,c.length);
  }
  return "";
}
var x = getCookie("operator_id")
if (x.length<10) {
    alert("登录状态失效，请重新登录~")
    top.location="login"
}