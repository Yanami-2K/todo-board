(function(){
var $=function(s){return document.querySelector(s);};
function esc(x){return String(x==null?"":x).replace(/[&<>"]/g,function(c){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c];});}
function rich(s){return esc(s).replace(/\*\*([^*]+)\*\*/g,"<b>$1</b>");}
var STAR='<svg viewBox="-10 -10 20 20"><path d="M0 -9 l3 7 7 1 -5 5 1 7 -6 -4 -6 4 1 -7 -5 -5 7 -1z" fill="#ff6f91" stroke="#fff8ef" stroke-width="2" stroke-linejoin="round"/></svg>';
var INFO=null, kind="human", MASCOT=null;
var params=new URLSearchParams(location.search);
if(params.get("token")) sessionStorage.setItem("tb_intake_token", params.get("token"));

function paint(){
  var name=INFO.name||"我";
  var latin=/^[\x00-\x7F]+$/.test(name);
  document.querySelectorAll(".me").forEach(function(x){x.textContent=name;});
  $("#lg").innerHTML=PETHEAD(INFO.look,"normal");
  $("#mascot").innerHTML=PET(MASCOT,"normal");
  $("#intro").innerHTML=rich(INFO.intro||"");
  $("#kb").textContent=INFO.hello||"";
  if(INFO.homepage) $("#home").href=INFO.homepage;
  document.title="给 "+name+" 登记一件事";
  var C=["c-pink","c-lav","c-mint","c-yellow","c-blue","c-pink","c-lav"];
  var cls=latin?"":" cn";
  if(Array.from(name).length>8) cls+=" fit";
  $("#name").innerHTML='<div class="tiles'+cls+'">'+Array.from(name).map(function(c,i){return '<span class="tile '+C[i%C.length]+'">'+esc(c)+'</span>';}).join("")+'</div>';
  var a='<span>给 '+esc(name)+' 登记一件事 '+STAR+'</span><span>REGISTER A TODO '+STAR+'</span><span>一天内回复 '+STAR+'</span><span>PROGRAMMER BY DAY '+STAR+'</span>';
  $("#t1").innerHTML=a+a+a+a;
  var b='<span>同事 · 外部 Agent · 朋友 ·</span><span>公开登记页 · INTAKE ·</span><span>先进收件箱，接受后才上看板 ·</span>';
  $("#t2").innerHTML=b+b+b+b;
  var tokenMode=INFO.mode==="token";
  $("#agenthint").textContent=tokenMode?"登记接口要带 INTAKE_TOKEN。你自己的 Agent 用它自己的 token 调 submit_to_intake 即可。":"公开登记不需要登录。Agent 可以调 API，或在 MCP 里用 submit_to_intake。";
  $("#code").innerHTML=tokenMode?'POST /api/intake\n<i>Authorization:</i> Bearer &lt;INTAKE_TOKEN&gt;\n{"title":"…","want":"10-16"}':'POST /api/intake\n{"title":"…","name":"…","want":"10-16"}';
  form();
}
function form(){
  var tokenMode=INFO.mode==="token";
  var saved=sessionStorage.getItem("tb_intake_token")||"";
  $("#fbox").innerHTML='<h2><div class="tiles sm"><span class="tile c-yellow">登</span><span class="tile c-paper">记</span></div><span style="font:700 26px/1.3 var(--f)">一件事</span></h2><p class="hint">写清楚想让他做什么就行，不用写很长。</p>'+
   '<form id="f"><span class="fl">我是</span><div class="who2"><button type="button" data-k="human" class="'+(kind==="human"?"on":"")+'">同事 / 朋友</button><button type="button" data-k="agent" class="'+(kind==="agent"?"on":"")+'">外部 Agent</button></div>'+
   (tokenMode?'<label class="fl" for="tk">登记 token</label><input class="inp" id="tk" name="token" value="'+esc(saved)+'" placeholder="部署时的 INTAKE_TOKEN" required>':"")+
   '<label class="fl" for="ti">想让他做什么</label><input class="inp big" id="ti" name="title" required placeholder="比如：帮忙看下 api-gateway 限流 PR #412">'+
   '<label class="fl" for="bd">补充说明 <small>选填</small></label><textarea class="inp" id="bd" name="body" placeholder="背景、链接、你希望的结果"></textarea>'+
   '<div class="row3"><div><label class="fl" for="nm">你的名字</label><input class="inp" id="nm" name="name" placeholder="张蔚"></div><div><label class="fl" for="ct">联系方式</label><input class="inp" id="ct" name="contact" placeholder="邮箱或 IM"></div><div><label class="fl" for="wt">希望在</label><input class="inp" id="wt" name="want" placeholder="10-16 前"></div></div>'+
   '<p class="merr" id="err"></p><div class="fbar"><button class="b ink lg" type="submit">交给 '+esc(INFO.name||"我")+' <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12L12 4M6 4h6v6"/></svg></button><p>提交后先进他的收件箱。</p></div></form>';
}
function burst(x,y){var b=document.createElement("div");b.className="burst";b.style.left=x+"px";b.style.top=y+"px";var Cc=["#ffd36e","#9fe3ff","#ff6f91","#cdf3e8","#e9e3ff","#ff9db5"],h="";for(var i=0;i<18;i++){var g=(i/18)*Math.PI*2,d=90+Math.random()*90,s=14+Math.random()*14,k=i%3;var shp=k===0?'<svg viewBox="-10 -10 20 20"><path d="M0 -9 l3 7 7 1 -5 5 1 7 -6 -4 -6 4 1 -7 -5 -5 7 -1z" fill="'+Cc[i%6]+'" stroke="#3b2f4a" stroke-width="2.5" stroke-linejoin="round"/></svg>':k===1?'<svg viewBox="-12 -12 24 20"><path d="M0 6 C-12 -2 -9 -12 -3 -11 C-1 -11 0 -9 0 -8 C0 -9 1 -11 3 -11 C9 -12 12 -2 0 6Z" fill="#ff6f91" stroke="#3b2f4a" stroke-width="2.5"/></svg>':'<svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="3.6" fill="'+Cc[(i+2)%6]+'" stroke="#3b2f4a" stroke-width="2"/></svg>';h+='<i style="--s:'+s+'px;--dx:'+Math.round(Math.cos(g)*d)+'px;--dy:'+Math.round(Math.sin(g)*d-30)+'px;--r:'+Math.round(Math.random()*240-120)+'deg">'+shp+'</i>';}b.innerHTML=h;document.body.appendChild(b);setTimeout(function(){b.remove();},1000);}
document.addEventListener("click",function(e){
  var k=e.target.closest("[data-k]");
  if(k){kind=k.dataset.k;document.querySelectorAll("[data-k]").forEach(function(x){x.classList.toggle("on",x===k);});}
  if(e.target.closest("[data-again]")) form();
});
document.addEventListener("submit",function(e){
  e.preventDefault();
  var f=e.target.elements;
  if(!f.title.value.trim()) return;
  var payload={title:f.title.value.trim(), body:f.body.value.trim(), name:f.name.value.trim(), contact:f.contact.value.trim(), want:f.want.value.trim(), kind:kind};
  if(f.token){ payload.token=f.token.value.trim(); sessionStorage.setItem("tb_intake_token", payload.token); }
  var btn=e.target.querySelector("[type=submit]");
  btn.disabled=true;
  fetch("/api/intake",{method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify(payload)})
    .then(function(r){ return r.json().then(function(d){ return {ok:r.ok, d:d}; }); })
    .then(function(x){
      btn.disabled=false;
      if(!x.ok){ var err=$("#err"); if(err) err.textContent=x.d.error||"没送出去"; return; }
      var box=btn.getBoundingClientRect();
      burst(box.left+box.width/2, box.top+box.height/2);
      $("#mascot").innerHTML=PET(MASCOT,"happy");
      $("#kb").textContent=INFO.thanks||"";
      var r=x.d;
      setTimeout(function(){
        $("#fbox").innerHTML='<div class="succ"><span class="pt">'+PET(MASCOT,"happy")+'</span><h3>已送达！</h3><p>「'+esc(r.title)+'」已经放进 '+esc(INFO.name)+' 的收件箱，<br>他看过之后会给你回邮件。</p><div class="ticket"><span class="tag">'+esc(r.id)+'</span>收件箱 · 待处理 · 刚刚</div><div style="margin-top:26px"><button class="b" data-again>再登记一件</button></div></div>';
        $("#fbox").scrollIntoView({behavior:"smooth",block:"center"});
      },350);
    })
    .catch(function(){ btn.disabled=false; var err=$("#err"); if(err) err.textContent="网络不行"; });
});
fetch("/api/intake").then(function(r){return r.json();}).then(function(info){
  INFO=info; MASCOT=info.look||PETS.gen(info.name||"");
  paint();
}).catch(function(){ $("#fbox").textContent="登记页加载失败"; });
})();
