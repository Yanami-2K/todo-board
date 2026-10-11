(function(){
var $=function(s,r){return(r||document).querySelector(s);};
var $$=function(s,r){return[].slice.call((r||document).querySelectorAll(s));};
function E(x){return String(x==null?"":x).replace(/[&<>"]/g,function(c){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;"}[c];});}
var S=null;
var IC={
 arrow:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12L12 4M6 4h6v6"/></svg>',
 left:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="M10 3L5 8l5 5"/></svg>',
 x:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round"><path d="M4 4l8 8M12 4l-8 8"/></svg>',
 check:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3.5 8.5l3 3 6-7"/></svg>',
 cv:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6l4 4 4-4"/></svg>',
 star:'<svg viewBox="-10 -10 20 20"><path d="M0 -9 l3 7 7 1 -5 5 1 7 -6 -4 -6 4 1 -7 -5 -5 7 -1z" fill="#ff6f91" stroke="#3b2f4a" stroke-width="2.5" stroke-linejoin="round"/></svg>',
 edit:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M10.5 3.5l2 2L6 12l-2.8.8L4 10z"/></svg>',
 dice:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linejoin="round"><rect x="2.5" y="2.5" width="11" height="11" rx="3"/><circle cx="5.8" cy="5.8" r="1" fill="currentColor" stroke="none"/><circle cx="10.2" cy="10.2" r="1" fill="currentColor" stroke="none"/><circle cx="8" cy="8" r="1" fill="currentColor" stroke="none"/></svg>',
 plus:'<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round"><path d="M8 3v10M3 8h10"/></svg>'
};
var MAILBOX='<svg class="mbx" viewBox="0 0 120 130"><rect x="52" y="70" width="14" height="58" rx="3" fill="#ffd6e2" stroke="#3b2f4a" stroke-width="4"/><path d="M14 30 Q14 8 40 8 L82 8 Q108 8 108 30 L108 76 L14 76Z" fill="#ff9db5" stroke="#3b2f4a" stroke-width="4" stroke-linejoin="round"/><path d="M40 8 Q66 8 66 30 L66 76" fill="none" stroke="#3b2f4a" stroke-width="4"/><rect x="26" y="40" width="28" height="7" rx="3.5" fill="#3b2f4a"/><g class="flag"><path d="M100 22 L100 -6 L118 -6 L118 8 L104 8" fill="#ff6f91" stroke="#3b2f4a" stroke-width="4" stroke-linejoin="round"/></g><g class="env"><rect x="22" y="14" width="38" height="26" rx="4" fill="#fffdf9" stroke="#3b2f4a" stroke-width="3.5" transform="rotate(-12 41 27)"/></g></svg>';
var TC=["c-pink","c-lav","c-yellow","c-mint","c-blue","c-pink","c-lav"];
var HERO_MAX=4;
var STX={todo:"待办",in_progress:"进行中",waiting:"等你确认",done:"已完成"};
var picks={}, agentPF={}, projSel="", tab="pending", openR=null, dwId=null, M=null, bornId=null;

function api(method, path, body){
  return fetch(path,{method:method, headers:body?{"Content-Type":"application/json"}:{}, body:body?JSON.stringify(body):undefined}).then(function(r){
    if(r.status===401){ location.href="/login"; throw new Error("先登录"); }
    return r.json().catch(function(){return {};}).then(function(d){
      if(!r.ok) throw new Error(d.error||"没成功");
      return d;
    });
  });
}
function load(keep){
  return fetch("/api/board").then(function(r){
    if(r.status===401){ location.href="/login"; throw new Error("auth"); }
    if(!r.ok) throw new Error("加载失败");
    return r.json();
  }).then(function(d){
    S=d;
    ["todos","agents","projects","questions","inbox","declined"].forEach(function(k){ if(!S[k]) S[k]=[]; });
    if(!S.activity) S.activity={};
    render(keep);
  }).catch(function(e){
    if(e.message!=="auth" && $("#view")) $("#view").innerHTML='<p class="lede">'+E(e.message||"加载失败")+'</p>';
  });
}
function todo(id){ return S.todos.filter(function(t){return t.id===id;})[0]; }
function agentById(id){ return S.agents.filter(function(a){return a.id===id;})[0]; }
function isAgent(id){ return !!agentById(id); }
function project(id){ return S.projects.filter(function(p){return p.id===id;})[0]; }
function nm(id){ if(!id||id==="owner") return "我自己"; var a=agentById(id); return a?a.name:id; }
function qOf(id){ return S.questions.filter(function(q){return q.agent_id===id;})[0]; }
function agentTodos(id){ return S.todos.filter(function(t){return t.delegate_id===id||t.lane===id;}); }
function currentTodo(a){
  var q=qOf(a.id); if(q) return todo(q.todo_id);
  var ts=agentTodos(a.id).filter(function(t){return t.status!=="done";});
  var ip=ts.filter(function(t){return t.status==="in_progress";}); if(ip[0]) return ip[0];
  var w=ts.filter(function(t){return t.status==="waiting";}); if(w[0]) return w[0];
  return ts[0]||null;
}
function mood(a){ if(!a||a.revoked) return "idle"; if(qOf(a.id)) return "ask"; var c=currentTodo(a); if(c&&c.status==="in_progress") return "working"; return "idle"; }
function sayOf(a){
  var m=mood(a);
  if(m==="ask") return trunc(qOf(a.id).text,16);
  if(m==="working"){ var c=currentTodo(a); var n=(c.steps||[]).length; if(n&&c.step) return trunc(c.steps[c.step-1]||"进行中",16); return "进行中"; }
  var ts=agentTodos(a.id);
  if(!ts.length) return "新来的，等你派活~";
  if(ts.every(function(t){return t.status==="done";})) return "都做完啦";
  return "有空，等你派活~";
}
function happySay(){ return "收到！"; }
function trunc(s,n){ s=String(s||""); var a=Array.from(s); return a.length>n?a.slice(0,n).join("")+"…":s; }
function colorHex(c){ return PETS.COLORS[c]||"#e9e3ff"; }
function agentColor(a){ return colorHex((a.look&&a.look.color)||"lav"); }
function pct(a){
  var t=currentTodo(a); if(!t) return 0;
  var n=(t.steps||[]).length;
  if(n&&t.step) return Math.max(8, Math.round((t.step-1)/n*100+(t.status==="in_progress"?8:0)));
  if(t.status==="in_progress") return 45;
  if(t.status==="waiting") return 30;
  return 8;
}
function stepLine(a){
  var t=currentTodo(a); if(!t) return "";
  var n=(t.steps||[]).length;
  if(n&&t.step) return "第 "+t.step+"/"+n+" 步 · "+(t.steps[t.step-1]||"");
  return STX[t.status]||"";
}
function tiles(txt,sm,latin){return '<div class="tiles'+(sm?" sm":"")+'">'+Array.from(txt).map(function(ch,i){return '<span class="tile '+TC[i%TC.length]+'"'+(latin?' style="font-family:Fredoka;font-weight:700"':"")+'>'+E(ch)+'</span>';}).join("")+'</div>';}
function shd(t,tag,latin,extra){return '<div class="shd">'+tiles(t,1,latin)+'<span class="ln"></span>'+(tag?'<span class="tag">'+tag+'</span>':"")+(extra||"")+'</div>';}
function petHTML(a,m,o){ return PET(a.look,m,o); }
function headHTML(a,m){ return PETHEAD(a.look,m); }
function meHead(m){ return PETHEAD(S.me.look,m||"normal"); }
function mePet(m,o){ return PET(S.me.look,m,o); }
function extIni(name){ name=name||"?"; if(/^[A-Za-z0-9_-]+$/.test(name)&&name.length<=4) return name.slice(0,2); return Array.from(name)[0]; }
function personAv(name, kind, sz){
  sz=sz||56; var ini=extIni(name); var bg=PETS.COLORS[PETS.COLOR_KEYS[PETS.hash(name||"?")%6]];
  var extra=kind==="agent"?";font-family:Fredoka;font-weight:700;border-radius:"+Math.round(sz*.32)+"px":"";
  return '<span class="av" style="width:'+sz+'px;height:'+sz+'px;font-size:'+Math.round(sz*.38)+'px;background:'+bg+extra+'">'+E(ini)+'</span>';
}
function avActor(kind,id,name,sz){
  sz=sz||56;
  if(kind==="owner"||id==="owner") return '<span class="pt mini" style="width:'+sz+'px;height:'+sz+'px;display:inline-block">'+meHead("normal")+'</span>';
  var a=id&&agentById(id);
  if(a) return '<span class="pt mini" style="width:'+sz+'px;height:'+sz+'px;display:inline-block">'+headHTML(a,mood(a))+'</span>';
  return personAv(name, kind==="extagent"||kind==="agent"?"agent":"human", sz);
}
function hasQ(t){ return S.questions.some(function(q){return q.todo_id===t.id;}); }
function stIcon(st){
  if(st==="done") return '<svg class="st" viewBox="0 0 30 30"><circle cx="15" cy="15" r="12" fill="#cdf3e8" stroke="#3b2f4a" stroke-width="3"/><path d="M9.5 15.5l3.5 3.5 7-8" fill="none" stroke="#3b2f4a" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/></svg>';
  if(st==="in_progress") return '<svg class="st" viewBox="0 0 30 30"><circle cx="15" cy="15" r="12" fill="#fffdf9" stroke="#3b2f4a" stroke-width="3"/><path d="M15 7 A8 8 0 0 1 15 23Z" fill="#ff6f91"/></svg>';
  if(st==="waiting") return '<svg class="st" viewBox="0 0 30 30"><circle cx="15" cy="15" r="12" fill="#ff6f91" stroke="#3b2f4a" stroke-width="3"/><text x="15" y="20.5" text-anchor="middle" font-family="Fredoka" font-weight="700" font-size="16" fill="#fff">?</text></svg>';
  return '<svg class="st" viewBox="0 0 30 30"><circle cx="15" cy="15" r="12" fill="#fffdf9" stroke="#3b2f4a" stroke-width="3" stroke-dasharray="4 3.2"/></svg>';
}
function ptag(pid){ var p=project(pid); return p?'<span class="ptag" style="background:'+colorHex(p.color)+'">'+E(p.name)+'</span>':""; }
function sortT(ts){ var o={in_progress:0,waiting:0,todo:1,done:2}; return ts.slice().sort(function(a,b){return (o[a.status]||9)-(o[b.status]||9);}); }
function route(){ var h=location.hash.replace(/^#\/?/,"").split("/"); if(h[0]==="agent"&&h[1]) return {name:"agent",id:decodeURIComponent(h[1])}; if(h[0]==="inbox") return {name:"inbox"}; if(h[0]==="projects") return {name:"projects",id:h[1]?decodeURIComponent(h[1]):undefined}; return {name:"home"}; }
function nav(r){
  var n=S.inbox.length, me=S.me.name;
  return '<a class="logo" href="#/"><span class="lg">'+meHead("normal")+'</span><b>'+E(me)+'<i>.</i></b><span style="font-weight:500;color:var(--ink2);font-size:15px;margin-left:4px">todo</span></a>'+
   '<div class="navr"><a class="np'+(r==="home"?" on":"")+'" href="#/">现在</a><a class="np'+(r==="projects"?" on":"")+'" href="#/projects">项目</a><a class="np'+(r==="inbox"?" on":"")+'" href="#/inbox">收件箱'+(n?'<span class="ct">'+n+'</span>':"")+'</a><a class="np" href="/intake">登记页 '+IC.arrow+'</a><button class="np" type="button" data-logout>退出</button></div>';
}
function decCount(){ return S.questions.length+S.inbox.length; }
function heroMeta(){
  var q=S.questions.length,i=S.inbox.length,n=q+i;
  var p=n?('有 '+(q?'<b>'+q+' 个 Agent</b> 在等你一句话':"")+(q&&i?"，还有 ":"")+(i?'<b>'+i+' 条新登记</b>等你接收':"")+'。'):'都处理完了，Agent 们正在各忙各的。';
  return '<div style="display:flex;align-items:flex-end;gap:22px;margin-top:30px">'+tiles("待确认")+'<span class="sticker num" id="cnt">'+n+'</span></div><p class="lede">'+p+'</p>';
}
var MO={ask:0,working:1,idle:2};
function heroPets(){
  var ks=S.agents.slice().sort(function(a,b){return MO[mood(a)]-MO[mood(b)];});
  var show=ks.length>HERO_MAX?ks.slice(0,HERO_MAX-1):ks, more=ks.length-show.length;
  var sz=show.length+(more?1:0)>3?136:168;
  return show.map(function(a){return '<a class="pt" href="#/agent/'+a.id+'" data-pet="'+a.id+'" style="display:block;width:'+sz+'px;height:'+sz+'px" title="'+E(a.name)+'">'+petHTML(a,mood(a))+'</a>';}).join("")+
    (more?'<a class="more" href="#agents" data-toagents style="margin-bottom:'+Math.round(sz*.2)+'px">+'+more+'<small>位</small></a>':"");
}
function opts(q){return q.options.map(function(o,i){return '<button class="b'+(i===q.options.length-1?" ink":"")+'" data-ans="'+q.id+'" data-opt="'+E(o)+'">'+E(o)+'</button>';}).join("");}
function qRow(q){
  var a=agentById(q.agent_id)||{name:"?",look:PETS.gen("?"),id:q.agent_id};
  return '<div class="ditem" data-q="'+q.id+'"><div><div class="drow" style="background:'+agentColor(a)+'">'+
   '<span class="pt" data-pet="'+a.id+'">'+headHTML(a,"ask")+'</span><div class="tx" data-open="'+q.todo_id+'"><div class="who">'+E(a.name)+'<span class="k">提问</span><span style="font-weight:500;color:var(--ink3)">'+E(q.ago)+'</span></div><div class="q">'+E(q.text)+'</div><div class="cx">'+E(q.context)+'</div></div>'+
   '<div class="acts">'+opts(q)+'</div></div></div></div>';
}
function assigneeOf(r){ var id=picks[r.id]||r.suggest_id||"owner"; if(id!=="owner"&&!isAgent(id)) id="owner"; return id; }
function rRow(r){
  var to=assigneeOf(r);
  return '<div class="ditem" data-r="'+r.id+'"><div><div class="drow" style="background:var(--paper)">'+
   personAv(r.from_name, r.from_kind, 62)+'<div class="tx" data-goinbox="'+r.id+'"><div class="who">'+E(r.from_name)+'<span class="k">新登记</span><span style="font-weight:500;color:var(--ink3)">'+E(r.via)+' · '+E(r.ago)+'</span></div><div class="q">'+E(r.title)+'</div><div class="cx">'+(r.want?"希望 "+E(r.want)+" · ":"")+'接受后交给 <b>'+E(nm(to))+'</b></div></div>'+
   '<div class="acts"><button class="b ghost" data-dec="'+r.id+'">拒绝</button><button class="b ink" data-acc="'+r.id+'">接受</button></div></div></div></div>';
}
function bigList(){
  if(!decCount()) return clearBox();
  return (S.questions.length?'<div class="grp-h"><span class="tag">AGENT 在问</span>'+S.questions.length+' 条，回一句它们就能继续</div>'+S.questions.map(qRow).join(""):"")+
   (S.inbox.length?'<div class="grp-h"><span class="tag">新登记</span>'+S.inbox.length+' 条，接受后才会上看板</div>'+S.inbox.map(rRow).join(""):"");
}
function clearBox(t1,t2){return '<div class="clear"><span class="pt">'+mePet("happy")+'</span><div><h3>'+(t1||"全部搞定！")+'</h3><p>'+(t2||"没有要你确认的事了。Agent 们会继续干活，有新情况我再叫你。")+'</p></div></div>';}
function todayList(){ return sortT(S.todos.filter(function(t){return t.lane==="self";})); }
function todayCard(){
  var ts=todayList();
  return '<div class="today">'+ts.map(function(t){var c=t.status==="done";return '<div class="trow'+(c?" checked":"")+'" data-t="'+t.id+'"><button class="cb'+(c?" on":"")+'" data-check="'+t.id+'" aria-label="完成">'+IC.check+'</button><div class="tt" data-open="'+t.id+'">'+E(t.title)+(t.note?' <span>· '+E(t.note)+'</span>':"")+'</div>'+ptag(t.project_id)+(t.due?'<span class="due'+(t.due==="今天"?" now":"")+'">'+E(t.due)+'</span>':"")+'</div>';}).join("")+
   '<form class="trow add" id="addform"><span class="cb dash"></span><input class="tt" name="title" maxlength="200" placeholder="记一件事…" autocomplete="off"></form></div>';
}
function agentCard(a){
  var m=mood(a), cur=currentTodo(a), say=m==="ask"?sayOf(a):sayOf(a);
  var liveCls=m==="ask"?"ask":m==="working"?"on":"";
  var liveTxt=a.revoked?"已吊销":{ask:"等你确认",idle:"空闲",working:"工作中"}[m];
  return '<a class="acard" href="#/agent/'+a.id+'" data-card="'+a.id+'" style="background:'+agentColor(a)+'"><div class="win">'+
   '<span class="live '+liveCls+'"><i></i>'+liveTxt+'</span>'+(say?'<span class="bub r" style="top:16px;right:12px">'+E(say)+'</span>':"")+
   '<span class="pt" data-pet="'+a.id+'">'+petHTML(a,m)+'</span></div>'+
   '<div class="nm"><b>'+E(a.name)+'</b><span>'+E(a.role)+'</span></div>'+
   (cur?'<div class="cur"><small>'+(m==="idle"||(cur.status==="waiting")?"在等":"正在做")+'</small>'+E(cur.title)+'</div><div class="pbar'+(m==="working"?"":" paused")+'"><i style="width:'+pct(a)+'%;--pf:'+agentColor(a)+'"></i></div><div class="pm"><span>'+E(stepLine(a))+'</span><span>'+E(a.last_label)+'</span></div>'
     :'<div class="cur"><small>还没有任务</small><span style="font-weight:500;color:var(--ink2)">在收件箱里接受登记时，可以直接交给它。</span></div><div class="pm"><span>加入于 '+E(a.since)+'</span><span></span></div>')+'</a>';
}
function agentCards(){
  return '<div class="agrid">'+S.agents.map(agentCard).join("")+
   '<button class="acard newc" data-newagent type="button"><span class="pt">'+PET(PETS.gen("?"),"idle")+'</span><b>新 Agent</b><span>起个名字，形象会自动生成</span></button></div>';
}
function agentsTag(){ var q=S.agents.filter(function(a){return mood(a)==="ask";}).length; return S.agents.length+" 位"+(q?" · "+q+" 位在等你":""); }
function home(){
  var ts=todayList();
  return '<section class="hero"><div><span class="eyebrow">'+IC.star+E(S.eyebrow||"")+'</span><div id="hmeta">'+heroMeta()+'</div></div><div class="pets" id="hpets">'+heroPets()+'</div></section>'+
   '<div class="big" id="big">'+bigList()+'</div>'+
   shd("今天", ts.length+" 件")+todayCard()+
   '<section class="band" id="agents-band">'+shd("Agents",'<span id="atag">'+agentsTag()+'</span>',true,'<button class="b" type="button" data-newagent style="margin-left:4px">'+IC.plus+'新 Agent</button>')+'<p class="sub">点进去看它的全部任务。</p><div id="agents">'+agentCards()+'</div></section>';
}
function pchips(todos,sel,key,withAdd,showAll){
  var cnt={}; todos.forEach(function(t){ if(t.project_id) cnt[t.project_id]=(cnt[t.project_id]||0)+1; });
  var ps=S.projects.filter(function(p){return showAll||cnt[p.id];});
  var none=todos.filter(function(t){return !t.project_id||!project(t.project_id);}).length;
  return '<div class="pchips"><button type="button" class="pc'+(!sel?" on":"")+'" data-pf="" data-key="'+key+'">全部 <span class="n">'+todos.length+'</span></button>'+
   ps.map(function(p){return '<button type="button" class="pc'+(sel===p.id?" on":"")+'" data-pf="'+p.id+'" data-key="'+key+'"><i style="background:'+colorHex(p.color)+'"></i>'+E(p.name)+' <span class="n">'+(cnt[p.id]||0)+'</span></button>';}).join("")+
   (none&&showAll?'<button type="button" class="pc'+(sel==="_none"?" on":"")+'" data-pf="_none" data-key="'+key+'"><i style="background:#fffdf9;border-style:dashed"></i>没归项目 <span class="n">'+none+'</span></button>':"")+
   (withAdd?'<button type="button" class="pc add" data-newproj>'+IC.plus+'新项目</button>':"")+'</div>';
}
function pfilter(todos,sel){ if(!sel) return todos; if(sel==="_none") return todos.filter(function(t){return !t.project_id||!project(t.project_id);}); return todos.filter(function(t){return t.project_id===sel;}); }
function tcard(t,showOwner){
  var who=showOwner?((t.delegate_id&&isAgent(t.delegate_id))?E(nm(t.delegate_id)):"我自己"):(t.source==="agent"?"它自己登记的":t.source==="self"?"我交给它的":"别人登记的");
  return '<div class="tcard'+(t.status==="done"?" done":"")+'" data-open="'+t.id+'">'+stIcon(hasQ(t)?"waiting":t.status)+'<div class="tt">'+E(t.title)+'<small>'+(hasQ(t)?"等你确认":STX[t.status])+(t.due?" · "+E(t.due):"")+" · "+who+'</small></div>'+ptag(t.project_id)+(showOwner&&t.delegate_id&&isAgent(t.delegate_id)?'<span class="pt" style="width:40px;height:40px;flex:none">'+headHTML(agentById(t.delegate_id),mood(agentById(t.delegate_id)))+'</span>':"")+'</div>';
}
function agentPage(id){
  var a=agentById(id);
  if(!a) return '<div class="big" style="margin-top:40px">'+clearBox("这个 Agent 已经不在了","它可能刚被删除。")+'</div>';
  var m=mood(a), q=qOf(a.id), cur=currentTodo(a);
  var ts=agentTodos(a.id); var open=ts.filter(function(t){return t.status!=="done";}), done=ts.filter(function(t){return t.status==="done";});
  var say=sayOf(a); var st=(cur&&cur.steps)||[]; var step=cur?cur.step:0;
  var list=sortT(pfilter(ts, agentPF[id]));
  var liveCls=m==="ask"?"ask":m==="working"?"on":"";
  var liveTxt=a.revoked?"已吊销":{ask:"等你确认",idle:"空闲",working:"工作中"}[m];
  var stepsHtml=st.length?'<ul class="steps">'+st.map(function(s,i){var c=i<step-1?"done":i===step-1?"cur":"";return '<li class="'+c+'"><span class="d">'+(c==="done"?IC.check:"")+'</span>'+E(s)+'</li>';}).join("")+'</ul>':"";
  return '<a class="b ghost" href="#/" style="margin-left:-14px">'+IC.left+'现在</a>'+
   '<section class="ahero"><div class="awin" style="background:'+agentColor(a)+'"><div class="in">'+
   '<span class="live '+liveCls+'"><i></i>'+liveTxt+'</span>'+(say?'<span class="bub r">'+E(say)+'</span>':"")+'<span class="pt" data-pet="'+a.id+'">'+petHTML(a,m)+'</span></div><span class="est">SINCE '+E(a.since_short||"")+'</span></div>'+
   '<div><span class="eyebrow" style="background:'+agentColor(a)+'">'+IC.star+'AGENT'+(a.role?" · "+E(a.role):"")+'</span><h1 class="aname" style="--sh-c:'+agentColor(a)+'">'+E(a.name)+'</h1><div class="arole"><span class="chip">'+ts.length+' 件任务</span><span class="chip">'+open.length+' 件没做完</span><span class="chip">最近活跃 '+E(a.last_label)+'</span><button class="b" type="button" data-editagent="'+a.id+'" style="height:36px;padding:0 14px;font-size:14px">'+IC.edit+'编辑</button></div>'+
   (q?'<div class="ask-box" style="margin-top:28px" data-qbox="'+q.id+'"><b>'+E(q.text)+'</b><p>'+E(q.context)+'</p><div class="acts">'+opts(q)+'</div></div>':"")+
   (cur?'<div class="nowc"><small>'+(m==="idle"?"在等":"正在做")+'</small><h3 data-open="'+cur.id+'">'+E(cur.title)+'</h3>'+stepsHtml+'</div>'
     :'<div class="nowc"><small>还没有任务</small><h3 style="cursor:default">等你派第一件活</h3><p style="margin:0;color:var(--ink2)">在收件箱接受登记时选它，或者在任意 todo 的详情里把「执行」改成它。</p></div>')+
   '</div></section>'+
   shd("全部任务", open.length+" 件进行中 · "+done.length+" 件完成")+
   (ts.length?pchips(ts,agentPF[id],"agent:"+id)+'<div class="tlist">'+list.map(function(t){return tcard(t);}).join("")+'</div>':'<p class="sub" style="margin-top:0">还没有任务。</p>')+
   '<div class="perm"><button type="button" data-perm><svg viewBox="0 0 20 20" width="22" height="22" fill="none" stroke="#3b2f4a" stroke-width="2.4" stroke-linejoin="round"><rect x="4" y="9" width="12" height="8" rx="2.5" fill="#fffdf9"/><path d="M7 9V7a3 3 0 0 1 6 0v2"/></svg>身份与权限<span style="font-weight:500;color:var(--ink2);font-size:14px">只能动自己泳道里的 todo</span><span class="cv">'+IC.cv+'</span></button><div class="bd"><div><dl><dt>Token</dt><dd>'+(a.revoked?"已吊销":E(a.token_prefix)+"••••")+'</dd><dt>作用域</dt><dd>自己泳道里的 todo：查看、新建、更新、完成、向你提问</dd><dt>接入方式</dt><dd>MCP · REST · Skill</dd><dt>启用于</dt><dd>'+E(a.since)+'</dd></dl><div class="rowb"><button type="button" class="b" data-rot="'+a.id+'">重新生成</button><button type="button" class="b ghost" data-revoke="'+a.id+'">吊销</button></div></div></div></div>';
}
function projects(pid){
  if(pid!==undefined) projSel=pid;
  if(projSel&&projSel!=="_none"&&!project(projSel)) projSel="";
  var all=S.todos.slice(); var list=sortT(pfilter(all, projSel)); var p=project(projSel);
  var h='<div class="ibhero"><div>'+tiles("项目")+'<p class="lede" style="max-width:640px">每条 todo 可以归到一个项目里。按项目看，自己的、Agent 的、别人登记的都在一起。</p></div><div class="pjstack">'+S.projects.slice(0,6).map(function(x,i){return '<span style="background:'+colorHex(x.color)+';--i:'+i+'">'+E(x.name)+'</span>';}).join("")+'</div></div>'+
   '<div style="margin-top:30px">'+pchips(all,projSel,"proj",true,true)+'</div>';
  if(p){
    var dn=list.filter(function(t){return t.status==="done";}).length, ag={};
    list.forEach(function(t){ if(t.delegate_id&&isAgent(t.delegate_id)) ag[t.delegate_id]=1; });
    h+='<div class="phead" style="background:'+colorHex(p.color)+'"><div><small>PROJECT</small><h2>'+E(p.name)+'</h2><div class="arole"><span class="chip">'+list.length+' 件</span><span class="chip">'+(list.length-dn)+' 件没做完</span>'+(Object.keys(ag).length?'<span class="chip" style="padding:2px 12px 2px 4px">'+Object.keys(ag).map(function(k){var a=agentById(k);return '<span class="pt" style="width:30px;height:30px;display:inline-block" title="'+E(a.name)+'">'+headHTML(a,mood(a))+'</span>';}).join("")+Object.keys(ag).length+' 个 Agent 在做</span>':"")+'</div></div><div class="pacts"><button class="b" type="button" data-editproj="'+p.id+'">'+IC.edit+'改名</button></div></div>';
  }
  h+=list.length?'<div class="tlist" style="margin-top:24px">'+list.map(function(t){return tcard(t,true);}).join("")+'</div>':'<div class="big" style="margin-top:24px">'+clearBox(p?"这个项目还是空的":"还没有 todo","打开任意 todo，在详情里把「项目」改成它。也可以先在「今天」里记一件。")+'</div>';
  return h;
}
function inbox(){
  var acc=S.todos.filter(function(t){return t.intake_id;});
  var h='<div class="ibhero"><div>'+tiles("收件箱")+'<p class="lede" style="max-width:640px">同事和外部 Agent 登记给你的事，先在这里过一眼。<b>接受</b>之后才会上看板，可以留给自己，也可以直接交给某个 Agent。</p></div><div class="mbox"><span class="bub">'+(S.inbox.length?S.inbox.length+' 封新的，拆开看看~':'都拆完啦')+'</span><span class="pt" data-pet="owner">'+mePet(S.inbox.length?"normal":"happy")+'</span>'+MAILBOX+'</div></div>'+
   '<div class="seg"><button type="button" data-tab="pending" class="'+(tab==="pending"?"on":"")+'">待处理 <span class="n">'+S.inbox.length+'</span></button><button type="button" data-tab="acc" class="'+(tab==="acc"?"on":"")+'">已接受 <span class="n">'+acc.length+'</span></button><button type="button" data-tab="dec" class="'+(tab==="dec"?"on":"")+'">已拒绝 <span class="n">'+S.declined.length+'</span></button></div><div class="ilist">';
  if(tab==="pending"){
    if(!S.inbox.length) h+='<div class="big" style="margin-top:0">'+clearBox("收件箱空啦","新的登记会出现在这里，也会在首页提醒你。")+'</div>';
    if(openR===null&&S.inbox[0]) openR=S.inbox[0].id;
    h+=S.inbox.map(function(r){
      var o=openR===r.id, to=assigneeOf(r);
      var tint=PETS.COLORS[PETS.COLOR_KEYS[PETS.hash(r.from_name||"")%6]];
      return '<div class="ditem" data-r="'+r.id+'"><div><div class="icard'+(o?" open":"")+'"><div class="ih" data-tog="'+r.id+'">'+personAv(r.from_name,r.from_kind,60)+'<div class="tx"><div class="who">'+E(r.from_name)+'<span style="font-weight:500;color:var(--ink3)">'+(r.from_kind==="agent"?"外部 Agent":"同事")+'</span></div><div class="ti">'+E(r.title)+'</div></div><span class="tm">'+E(r.ago)+'</span></div>'+
       '<div class="xb"><div><div class="xin"><div class="say" style="background:'+tint+'">'+E(r.body||"（没有补充说明）")+'</div><div class="facts">'+(r.want?'<span class="chip">希望 '+E(r.want)+'</span>':"")+'<span class="chip">'+E(r.via)+'</span>'+(r.contact?'<span class="chip" style="font-weight:500">'+E(r.contact)+'</span>':"")+'</div>'+
       '<div class="lbl">交给谁</div><div class="opts">'+["owner"].concat(S.agents.map(function(a){return a.id;})).map(function(k){return '<button type="button" class="opt'+(to===k?" on":"")+'" data-pick="'+r.id+'" data-to="'+k+'"><span class="pt">'+(k==="owner"?meHead("normal"):headHTML(agentById(k),"normal"))+'</span>'+E(nm(k))+(r.suggest_id===k||(!r.suggest_id&&k==="owner")?'<span class="sg">建议</span>':"")+'</button>';}).join("")+'</div>'+
       '<div class="iact"><button class="b ink lg" type="button" data-acc="'+r.id+'">'+IC.check+'接受，交给 '+E(nm(to))+'</button><button class="b ghost lg" type="button" data-dec="'+r.id+'">拒绝</button><span style="margin-left:auto;font-size:14px;color:var(--ink3)">对方会收到一封通知邮件</span></div></div></div></div></div></div></div>';
    }).join("");
  } else if(tab==="acc"){
    h+=acc.map(function(t){return '<div class="arow" data-open="'+t.id+'">'+avActor(t.creator_kind,t.creator_id,t.creator_name,46)+'<div class="tt">'+E(t.title)+'<small>'+E(t.creator_name)+' 登记 · 交给 '+E(nm(t.delegate_id||"owner"))+'</small></div>'+stIcon(t.status)+'</div>';}).join("")||'<p class="sub">还没有接受过登记。</p>';
  } else {
    h+=S.declined.length?S.declined.map(function(r){return '<div class="arow" style="opacity:.6">'+personAv(r.from_name,r.from_kind,46)+'<div class="tt">'+E(r.title)+'<small>'+E(r.from_name)+' · 已拒绝</small></div></div>';}).join(""):'<p class="sub" style="margin-top:10px">还没有拒绝过任何登记。</p>';
  }
  return h+'</div>';
}
function kindLabel(k){ return {owner:"我",agent:"Agent",external:"同事",extagent:"外部 Agent",human:"同事"}[k]||""; }
function descHTML(s){ if(!s) return '<span style="color:var(--ink3)">没有写说明。</span>'; return E(s).replace(/`([^`]+)`/g,"<code>$1</code>"); }
function drawer(id){
  var t=todo(id); if(!t) return;
  var own=t.delegate_id&&isAgent(t.delegate_id)?agentById(t.delegate_id):null;
  var q=S.questions.filter(function(x){return x.todo_id===id;})[0];
  var col=own?agentColor(own):(t.project_id&&project(t.project_id)?colorHex(project(t.project_id).color):t.lane==="others"?"#ffd6e2":"#e9e3ff");
  var srcT={self:"我自己建的",agent:"Agent 自己登记",intake_form:"公开登记页",intake_api:"登记 API"}[t.source]||t.source;
  var act=S.activity[id]||[];
  var p=project(t.project_id);
  $("#dw").innerHTML='<div class="dw-top" style="background-color:'+col+'">'+(own?'<span class="pt" data-pet="'+own.id+'">'+petHTML(own,mood(own),{noShadow:true})+'</span>':avActor(t.creator_kind,t.creator_id,t.creator_name,72))+'<div class="tx"><span class="chip" style="gap:8px">'+stIcon(q?"waiting":t.status).replace('class="st"','class="st" style="width:20px;height:20px"')+(q?"等你确认":STX[t.status])+'</span><h2>'+E(t.title)+'</h2></div><button class="x" type="button" data-close aria-label="关闭">'+IC.x+'</button></div>'+
   '<div class="dw-b">'+(q?'<div class="ask-box" data-qbox="'+q.id+'"><b>'+E(q.text)+'</b><p>'+E(q.context)+'</p><div class="acts">'+opts(q)+'</div></div>':"")+
   '<dl class="props"><dt>项目</dt><dd><button type="button" class="pick" data-menu="proj" data-t="'+t.id+'">'+(p?'<i style="background:'+colorHex(p.color)+'"></i>'+E(p.name):'<span style="color:var(--ink3)">没归项目</span>')+IC.cv+'</button></dd>'+
   '<dt>执行</dt><dd><button type="button" class="pick" data-menu="exec" data-t="'+t.id+'">'+(own?'<span class="pt mini">'+headHTML(own,mood(own))+'</span>'+E(own.name):avActor("owner","owner",S.me.name,30)+'我自己')+IC.cv+'</button></dd><dt>创建者</dt><dd>'+avActor(t.creator_kind,t.creator_id,t.creator_name,30)+E(t.creator_name)+'<span class="kd">'+kindLabel(t.creator_kind)+'</span></dd><dt>来源</dt><dd>'+srcT+'</dd><dt>截止</dt><dd>'+(t.due?E(t.due):'<span style="color:var(--ink3);font-weight:500">没设</span>')+'</dd></dl>'+
   '<div class="dsec">说明</div><div class="desc">'+descHTML(t.description)+'</div>'+
   '<div class="dsec">动态</div><ul class="feed">'+act.map(function(x){return '<li>'+avActor(x.actor_kind,x.actor_id,x.actor_name,36)+'<div class="ln"><span class="w">'+E(x.actor_name)+'</span><span class="tm">'+E(x.ago)+'</span>'+(x.kind==="m"?'<div class="bb">'+E(x.text)+'</div>':'<div style="color:var(--ink2)">'+E(x.text)+'</div>')+'</div></li>';}).join("")+'</ul>'+
   '<div class="mf"><span>'+E(t.id)+'</span></div></div>'+
   '<form class="compose" id="cform" data-todo="'+t.id+'"><input name="body" maxlength="2000" placeholder="'+(own?'给 '+E(own.name)+' 留句话…':'写点备注…')+'" autocomplete="off"><button class="b ink" type="submit">发送</button></form>';
  $("#dw").classList.add("on"); $("#scrim").classList.add("on"); dwId=id;
}
function closeDw(){ $("#dw").classList.remove("on"); $("#scrim").classList.remove("on"); dwId=null; closeMenu(); }
function closeMenu(){ var m=$("#menu"); if(m) m.remove(); }
function openMenu(btn){
  closeMenu(); var t=todo(btn.dataset.t); if(!t) return; var kind=btn.dataset.menu, items;
  if(kind==="proj") items=S.projects.map(function(p){return {v:p.id,on:t.project_id===p.id,h:'<i style="background:'+colorHex(p.color)+'"></i>'+E(p.name)};}).concat([{v:"",on:!t.project_id,h:'<i style="background:#fffdf9;border-style:dashed"></i>不归项目'},{v:"_new",h:IC.plus+'新项目…',cls:"add"}]);
  else items=["owner"].concat(S.agents.map(function(a){return a.id;})).map(function(k){ var on=k==="owner"?!t.delegate_id:t.delegate_id===k; return {v:k,on:on,h:(k==="owner"?avActor("owner","owner",S.me.name,34):'<span class="pt">'+headHTML(agentById(k),"normal")+'</span>')+E(nm(k))}; });
  var m=document.createElement("div"); m.id="menu"; m.className="menu";
  m.innerHTML=items.map(function(it){return '<button type="button" class="'+(it.on?"on ":"")+(it.cls||"")+'" data-mv="'+E(it.v)+'">'+it.h+(it.on?'<span class="ok">'+IC.check+'</span>':"")+'</button>';}).join("");
  m.dataset.kind=kind; m.dataset.t=t.id; document.body.appendChild(m);
  var r=btn.getBoundingClientRect(); m.style.left=Math.max(8,r.left)+"px"; var top=r.bottom+8; if(top+m.offsetHeight>innerHeight-12) top=Math.max(12,r.top-8-m.offsetHeight); m.style.top=top+"px";
}
function closeModal(){
  var token=M&&M.type==="token", id=bornId;
  $("#modal").classList.remove("on"); $("#modal").innerHTML=""; M=null; bornId=null;
  if(token){ load(true).then(function(){ if(!id) return; var c=$('[data-card="'+id+'"]'); if(!c) return; c.scrollIntoView({block:"center",behavior:"smooth"}); c.classList.add("born"); var p=c.querySelector("[data-pet]"); var a=agentById(id); if(!a||!p) return; p.innerHTML=petHTML(a,"happy"); burstAt(p,16); setTimeout(function(){ if(agentById(id)) p.innerHTML=petHTML(agentById(id),mood(agentById(id))); },1700); }); }
}
function moodBtns(){ return [["working","工作中"],["ask","等你确认"],["idle","空闲"],["happy","处理完"]].map(function(x){return '<button type="button" class="'+(M.mood===x[0]?"on":"")+'" data-mm="'+x[0]+'">'+x[1]+'</button>';}).join(""); }
function paintPrev(){ if(!M||M.type!=="agent") return; $("#mpet").innerHTML=PET(M.look,M.mood); $("#mprev").style.background=colorHex(M.look.color); $("#mcombo").textContent=PETS.describe(M.look); $("#mmoods").innerHTML=moodBtns(); }
function agentModal(id){
  var a=id?agentById(id):null;
  M={type:"agent",id:id||null,salt:0,custom:!!a,look:a?a.look:PETS.gen(""),mood:"working",del:false};
  $("#modal").innerHTML='<div class="mcard"><div class="mprev" id="mprev"><span class="live on"><i></i>预览</span><span class="pt" id="mpet"></span><div class="moods" id="mmoods">'+moodBtns()+'</div></div>'+
   '<form class="mform" id="mform" autocomplete="off"><div class="mh">'+tiles(id?"编辑":"新伙伴",1)+'<button type="button" class="x" data-mclose aria-label="关闭">'+IC.x+'</button></div>'+
   '<label class="fl" for="an">名字</label><input class="inp big" id="an" maxlength="24" placeholder="比如 my-agent" value="'+(a?E(a.name):"")+'">'+
   '<label class="fl" for="ar">负责什么 <small>选填</small></label><input class="inp" id="ar" maxlength="30" placeholder="比如 跑集成测试" value="'+(a?E(a.role):"")+'">'+
   '<span class="fl">形象</span><div class="lookline"><span class="combo" id="mcombo"></span><button type="button" class="b" data-reroll>'+IC.dice+'换一个</button><button type="button" class="b ghost" data-relook>按名字生成</button></div>'+
   '<p class="mhint">按名字自动生成：同一个名字永远是同一个样子，不喜欢就换一个。</p><p class="merr" id="merr"></p>'+
   '<div class="mfoot"><button class="b ink lg" type="submit">'+(id?"保存":IC.plus+"创建")+'</button><button class="b ghost lg" type="button" data-mclose>取消</button>'+(id?'<button class="b danger" type="button" data-delagent="'+id+'">删除这个 Agent</button>':"")+'</div></form></div>';
  $("#modal").classList.add("on"); paintPrev(); setTimeout(function(){ var n=$("#an"); if(n) n.focus(); },60);
}
function showToken(id, token, born){
  M={type:"token"}; if(born) bornId=id;
  $("#modal").innerHTML='<div class="mcard sm"><div class="mform"><div class="mh">'+tiles("令牌",1)+'<button type="button" class="x" data-mclose aria-label="关闭">'+IC.x+'</button></div>'+
   '<p class="lede" style="margin-top:8px">这串 token 只显示一次。交给这个 Agent，丢了就再生成一串，旧的会失效。</p>'+
   '<input class="inp tokbox" id="tok" readonly value="'+E(token)+'">'+
   '<p class="mhint">请求头 <b>Authorization: Bearer</b> 这串 token。MCP 和 REST 用的是同一个。</p>'+
   '<div class="mfoot"><button class="b ink lg" type="button" data-copy>复制</button><button class="b ghost lg" type="button" data-mclose>我已记下</button></div></div></div>';
  $("#modal").classList.add("on");
}
var SW=PETS.COLOR_KEYS;
function projModal(id){
  var p=id?project(id):null;
  M={type:"proj",id:id||null,color:p?p.color:SW[0],del:false,touched:!!p,after:M&&M.after};
  $("#modal").innerHTML='<div class="mcard sm"><form class="mform" id="mform" autocomplete="off" style="padding-top:28px"><div class="mh">'+tiles(id?"改名":"新项目",1)+'<button type="button" class="x" data-mclose aria-label="关闭">'+IC.x+'</button></div>'+
   '<label class="fl" for="pn">项目名</label><input class="inp big" id="pn" maxlength="24" placeholder="比如 新项目" value="'+(p?E(p.name):"")+'">'+
   '<span class="fl">颜色</span><div class="sw" id="psw">'+SW.map(function(c){return '<button type="button" data-sw="'+c+'" class="'+(M.color===c?"on":"")+'" style="background:'+colorHex(c)+'" aria-label="'+PETS.COLOR_ZH[c]+'"></button>';}).join("")+'</div><p class="merr" id="merr"></p>'+
   '<div class="mfoot"><button class="b ink lg" type="submit">'+(id?"保存":IC.plus+"创建")+'</button><button class="b ghost lg" type="button" data-mclose>取消</button>'+(id?'<button class="b danger" type="button" data-delproj="'+id+'">删除项目</button>':"")+'</div></form></div>';
  $("#modal").classList.add("on"); setTimeout(function(){ var n=$("#pn"); if(n) n.focus(); },60);
}
function burst(x,y,n){var b=document.createElement("div");b.className="burst";b.style.left=x+"px";b.style.top=y+"px";var C=["#ffd36e","#9fe3ff","#ff6f91","#cdf3e8","#e9e3ff","#ff9db5"];var h="";n=n||14;for(var i=0;i<n;i++){var ang=(i/n)*Math.PI*2+Math.random()*.4,d=70+Math.random()*70,s=14+Math.random()*12,k=i%3;var shp=k===0?'<svg viewBox="-10 -10 20 20"><path d="M0 -9 l3 7 7 1 -5 5 1 7 -6 -4 -6 4 1 -7 -5 -5 7 -1z" fill="'+C[i%6]+'" stroke="#3b2f4a" stroke-width="2.5" stroke-linejoin="round"/></svg>':k===1?'<svg viewBox="-12 -12 24 20"><path d="M0 6 C-12 -2 -9 -12 -3 -11 C-1 -11 0 -9 0 -8 C0 -9 1 -11 3 -11 C9 -12 12 -2 0 6Z" fill="#ff6f91" stroke="#3b2f4a" stroke-width="2.5"/></svg>':'<svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="3.6" fill="'+C[(i+2)%6]+'" stroke="#3b2f4a" stroke-width="2"/></svg>';h+='<i style="--s:'+s+'px;--dx:'+Math.round(Math.cos(ang)*d)+'px;--dy:'+Math.round(Math.sin(ang)*d-30)+'px;--r:'+Math.round(Math.random()*240-120)+'deg">'+shp+'</i>';}b.innerHTML=h;document.body.appendChild(b);setTimeout(function(){b.remove();},1000);}
function burstAt(el,n){ if(!el) return; var r=el.getBoundingClientRect(); burst(r.left+r.width/2,r.top+r.height/2,n); }
function stamp(row,txt,col){ var s=document.createElement("span"); s.className="done-stamp"; s.textContent=txt; if(col) s.style.background=col; if(col==="#e9e3ff") s.style.color="var(--ink)"; row.appendChild(s); }
function setPets(id,m){ var a=agentById(id); if(!a) return; $$('[data-pet="'+id+'"]').forEach(function(el){ var keep=!el.closest(".drow")&&!el.closest(".dw-top"); el.innerHTML=keep?petHTML(a,m):headHTML(a,m); }); }
var toastT;
function toast(html){ var t=$("#toast"); if(!t) return; t.innerHTML='<span class="pt">'+meHead("happy")+'</span><span>'+html+'</span>'; t.classList.add("on"); clearTimeout(toastT); toastT=setTimeout(function(){t.classList.remove("on");},3800); }
function celebrate(row,btn,kind){
  return new Promise(function(resolve){
    if(btn) btn.style.pointerEvents="none";
    if(kind!=="dec"&&btn) burstAt(btn);
    var card=row.querySelector(".drow,.icard,.ask-box")||row;
    stamp(card, kind==="ans"?"收到！":kind==="acc"?"收下！":"下次吧", kind==="dec"?"#e9e3ff":null);
    setTimeout(function(){ row.classList.add("gone"); }, kind==="dec"?650:950);
    setTimeout(resolve, kind==="dec"?1150:1450);
  });
}
function render(keep){
  if(!S) return;
  var r=route();
  var keepDw=dwId;
  $("#nav").innerHTML=nav(r.name);
  var v=r.name==="agent"?agentPage(r.id):r.name==="inbox"?inbox():r.name==="projects"?projects(r.id):home();
  $("#view").innerHTML='<div class="view">'+v+'</div>';
  document.title=S.me.name+" · todo";
  if(!keep) window.scrollTo(0,0);
  if(keepDw&&todo(keepDw)) drawer(keepDw);
  else closeDw();
}
window.addEventListener("hashchange",function(){ closeDw(); if(M&&M.type!=="token") closeModal(); load(false); });
document.addEventListener("input",function(e){
  if(!M) return;
  if(e.target.id==="an"&&M.type==="agent"){ if(!M.custom){ M.look=PETS.gen(e.target.value.trim()||""); paintPrev(); } $("#merr").textContent=""; }
  if(e.target.id==="pn"&&M.type==="proj"&&!M.touched){ var v=e.target.value.trim(); M.color=SW[PETS.hash(v)%6]; $$("#psw button").forEach(function(b){b.classList.toggle("on",b.dataset.sw===M.color);}); }
});
document.addEventListener("submit",function(e){
  if(e.target.id==="addform"){
    e.preventDefault();
    var title=e.target.elements.title.value.trim();
    if(!title) return;
    api("POST","/api/todos",{title:title}).then(function(){ return load(true); }).then(function(){ toast("记下了"); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(e.target.id==="cform"){
    e.preventDefault();
    var body=e.target.elements.body.value.trim();
    if(!body) return;
    var id=e.target.dataset.todo;
    api("POST","/api/todos/"+id+"/comments",{body:body}).then(function(){ return load(true); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(!M||e.target.id!=="mform") return;
  e.preventDefault();
  if(M.type==="agent"){
    var name=$("#an").value.trim(), role=$("#ar").value.trim();
    if(!name){ $("#merr").textContent="先起个名字吧"; return; }
    var look=M.look, editing=M.id;
    var req=editing?api("PATCH","/api/agents/"+editing,{name:name,role:role,look:look}):api("POST","/api/agents",{name:name,role:role,look:look});
    req.then(function(d){
      if(editing){ M=null; $("#modal").classList.remove("on"); $("#modal").innerHTML=""; load(true).then(function(){ toast("已保存 "+E(name)); var pe=$('.awin [data-pet="'+editing+'"]'); if(pe){ var a=agentById(editing); pe.innerHTML=petHTML(a,"happy"); burstAt(pe,12); setTimeout(function(){ if(agentById(editing)) pe.innerHTML=petHTML(agentById(editing),mood(agentById(editing))); },1500);} }); return; }
      showToken(d.agent.id, d.token, true);
    }).catch(function(err){ $("#merr").textContent=err.message; });
    return;
  }
  if(M.type==="proj"){
    var pn=$("#pn").value.trim();
    if(!pn){ $("#merr").textContent="项目名不能空着"; return; }
    var color=M.color, id=M.id, after=M.after;
    var reqp=id?api("PATCH","/api/projects/"+id,{name:pn,color:color}):api("POST","/api/projects",{name:pn,color:color});
    reqp.then(function(p){
      M=null; $("#modal").classList.remove("on"); $("#modal").innerHTML="";
      var done=Promise.resolve();
      if(!id&&after) done=api("PATCH","/api/todos/"+after,{project_id:p.id});
      return done.then(function(){ if(!id&&!after) projSel=p.id; return load(true); }).then(function(){ toast(id?"项目已改名":"新项目 <b>"+E(pn)+"</b> 建好了"); if(!id&&!after&&route().name!=="projects") location.hash="#/projects"; });
    }).catch(function(err){ $("#merr").textContent=err.message; });
  }
});
document.addEventListener("click",function(e){
  var mi=e.target.closest("#menu [data-mv]");
  if(mi){
    var mn=$("#menu"), tid=mn.dataset.t, v=mi.dataset.mv; closeMenu();
    if(mn.dataset.kind==="proj"){
      if(v==="_new"){ M={after:tid}; projModal(); return; }
      api("PATCH","/api/todos/"+tid,{project_id:v}).then(function(){return load(true);}).catch(function(err){toast(E(err.message));});
    } else {
      api("PATCH","/api/todos/"+tid,{delegate_id:v==="owner"?"":v}).then(function(){return load(true);}).catch(function(err){toast(E(err.message));});
    }
    return;
  }
  if(!e.target.closest("#menu,[data-menu]")) closeMenu();
  var t=e.target.closest("[data-ans],[data-acc],[data-dec],[data-check],[data-open],[data-close],[data-tab],[data-tog],[data-pick],[data-perm],[data-goinbox],[data-menu],[data-newagent],[data-editagent],[data-mclose],[data-reroll],[data-relook],[data-mm],[data-delagent],[data-pf],[data-newproj],[data-editproj],[data-delproj],[data-sw],[data-toagents],[data-logout],[data-rot],[data-revoke],[data-copy],#scrim,#modal");
  if(!t) return;
  if(t.id==="modal"){ if(e.target===t) closeModal(); return; }
  if(t.id==="scrim"||t.hasAttribute("data-close")) return closeDw();
  if(t.hasAttribute("data-mclose")) return closeModal();
  if(t.hasAttribute("data-copy")){ var inp=$("#tok"); if(!inp) return; inp.select(); var ok=false; try{ ok=document.execCommand("copy"); }catch(err){} if(navigator.clipboard) navigator.clipboard.writeText(inp.value).then(function(){ toast("已复制"); },function(){ if(ok) toast("已复制"); }); else if(ok) toast("已复制"); return; }
  if(t.hasAttribute("data-logout")){ fetch("/api/logout",{method:"POST"}).finally(function(){ location.href="/login"; }); return; }
  if(t.hasAttribute("data-menu")){ var open=$("#menu")&&$("#menu").dataset.kind===t.dataset.menu&&$("#menu").dataset.t===t.dataset.t; if(open) closeMenu(); else openMenu(t); return; }
  if(t.hasAttribute("data-newagent")){ e.preventDefault(); agentModal(); return; }
  if(t.hasAttribute("data-editagent")){ agentModal(t.dataset.editagent); return; }
  if(t.hasAttribute("data-reroll")){ M.salt++; M.custom=true; M.look=PETS.gen($("#an").value.trim(), M.salt); paintPrev(); return; }
  if(t.hasAttribute("data-relook")){ M.salt=0; M.custom=false; M.look=PETS.gen($("#an").value.trim()); paintPrev(); return; }
  if(t.hasAttribute("data-mm")){ M.mood=t.dataset.mm; paintPrev(); return; }
  if(t.hasAttribute("data-delagent")){
    var did=t.dataset.delagent, n=agentTodos(did).length;
    if(!M.del){ M.del=true; t.textContent="再点一次确认删除"+(n?"（"+n+" 件任务转回给你）":""); t.classList.add("arm"); return; }
    var dn=agentById(did).name;
    api("DELETE","/api/agents/"+did).then(function(){ M=null; $("#modal").classList.remove("on"); $("#modal").innerHTML=""; location.hash="#/"; return load(false); }).then(function(){ toast("已删除 "+E(dn)); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(t.hasAttribute("data-newproj")){ projModal(); return; }
  if(t.hasAttribute("data-editproj")){ projModal(t.dataset.editproj); return; }
  if(t.hasAttribute("data-sw")){ M.color=t.dataset.sw; M.touched=true; $$("#psw button").forEach(function(b){b.classList.toggle("on",b===t);}); return; }
  if(t.hasAttribute("data-delproj")){
    var dp=t.dataset.delproj, pc=S.todos.filter(function(x){return x.project_id===dp;}).length;
    if(!M.del){ M.del=true; t.textContent="再点一次确认删除"+(pc?"（"+pc+" 件 todo 会变成没归项目）":""); t.classList.add("arm"); return; }
    var pnm=project(dp).name;
    api("DELETE","/api/projects/"+dp).then(function(){ M=null; $("#modal").classList.remove("on"); $("#modal").innerHTML=""; projSel=""; return load(true); }).then(function(){ toast("已删除项目 "+E(pnm)); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(t.hasAttribute("data-pf")){ var key=t.dataset.key, v2=t.dataset.pf; if(key==="proj"){ projSel=v2; if(route().name==="projects") history.replaceState(null,"","#/projects"+(v2?"/"+encodeURIComponent(v2):"")); } else agentPF[key.slice(6)]=v2; render(true); return; }
  if(t.hasAttribute("data-toagents")){ e.preventDefault(); var band=$("#agents-band"); if(band) band.scrollIntoView({behavior:"smooth"}); return; }
  if(t.hasAttribute("data-rot")){
    api("POST","/api/agents/"+t.dataset.rot+"/token",{}).then(function(d){ showToken(d.agent.id, d.token, false); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(t.hasAttribute("data-revoke")){
    if(!t.dataset.arm){ t.dataset.arm="1"; t.textContent="再点一次确认吊销"; return; }
    api("POST","/api/agents/"+t.dataset.revoke+"/revoke",{}).then(function(){ return load(true); }).then(function(){ toast("token 已吊销"); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(t.hasAttribute("data-ans")){
    e.preventDefault();
    var qid=t.dataset.ans, opt=t.dataset.opt, q=S.questions.filter(function(x){return x.id===qid;})[0];
    if(!q) return;
    var a=agentById(q.agent_id);
    if(a){ setPets(a.id,"happy"); $$('[data-pet="'+a.id+'"]').forEach(function(p){ var b=p.parentNode.querySelector(".bub"); if(b) b.textContent=happySay(); }); }
    var row=t.closest(".ditem")||t.closest(".ask-box");
    var done=function(){ return api("POST","/api/questions/"+qid+"/answer",{option:opt}).then(function(){ return load(true); }).then(function(){ toast("已回复 "+E(a?a.name:"")+"：<b>"+E(opt)+"</b>"); }); };
    if(row&&row.classList.contains("ditem")) celebrate(row,t,"ans").then(done).catch(function(err){ toast(E(err.message)); load(true); });
    else { if(t) burstAt(t); if(row) stamp(row,"收到！"); setTimeout(function(){ done().catch(function(err){ toast(E(err.message)); load(true); }); },700); }
    return;
  }
  if(t.hasAttribute("data-acc")){
    var rid=t.dataset.acc, r=S.inbox.filter(function(x){return x.id===rid;})[0]; if(!r) return;
    var to=assigneeOf(r); var row2=t.closest(".ditem");
    celebrate(row2,t,"acc").then(function(){ return api("POST","/api/inbox/"+rid+"/accept",{assignee:to}); }).then(function(d){ if(openR===rid) openR=null; return load(route().name==="inbox").then(function(){ toast('已接受，交给 '+E(nm(to))); }); }).catch(function(err){ toast(E(err.message)); load(true); });
    return;
  }
  if(t.hasAttribute("data-dec")){
    var rid2=t.dataset.dec, row3=t.closest(".ditem");
    celebrate(row3,t,"dec").then(function(){ return api("POST","/api/inbox/"+rid2+"/decline",{}); }).then(function(){ if(openR===rid2) openR=null; return load(route().name==="inbox"); }).then(function(){ toast("已拒绝"); }).catch(function(err){ toast(E(err.message)); load(true); });
    return;
  }
  if(t.hasAttribute("data-check")){
    var td=todo(t.dataset.check); if(!td) return;
    var next=td.status==="done"?"todo":"done";
    api("PATCH","/api/todos/"+td.id,{status:next}).then(function(updated){ td.status=updated.status; t.classList.toggle("on",updated.status==="done"); t.closest(".trow").classList.toggle("checked",updated.status==="done"); if(updated.status==="done") burstAt(t); }).catch(function(err){ toast(E(err.message)); });
    return;
  }
  if(t.hasAttribute("data-open")){ e.preventDefault(); drawer(t.dataset.open); return; }
  if(t.hasAttribute("data-goinbox")){ openR=t.dataset.goinbox; tab="pending"; if(location.hash.replace(/\/$/,"")==="#/inbox") render(true); else location.hash="#/inbox"; return; }
  if(t.hasAttribute("data-tab")){ tab=t.dataset.tab; render(true); return; }
  if(t.hasAttribute("data-tog")){ var id=t.dataset.tog; openR=openR===id?"":id; $$(".icard").forEach(function(c){ var me=c.querySelector("[data-tog]").dataset.tog; c.classList.toggle("open",me===openR); }); return; }
  if(t.hasAttribute("data-pick")){
    picks[t.dataset.pick]=t.dataset.to;
    var card=t.closest(".icard")||t.closest(".drow");
    if(card){ $$(".opt",card).forEach(function(o){o.classList.toggle("on",o===t);}); var acc=card.querySelector("[data-acc]"); if(acc) acc.innerHTML=IC.check+"接受，交给 "+E(nm(t.dataset.to)); }
    var homeRow=t.closest(".ditem");
    if(homeRow){ var cx=homeRow.querySelector(".cx"); if(cx){ var rr=S.inbox.filter(function(x){return x.id===t.dataset.pick;})[0]; cx.innerHTML=(rr&&rr.want?"希望 "+E(rr.want)+" · ":"")+'接受后交给 <b>'+E(nm(t.dataset.to))+'</b>'; } }
    return;
  }
  if(t.hasAttribute("data-perm")){ t.parentNode.classList.toggle("open"); return; }
});
document.addEventListener("keydown",function(e){ if(e.key==="Escape"){ if(M) closeModal(); else if($("#menu")) closeMenu(); else closeDw(); }});
window.addEventListener("scroll",closeMenu,{passive:true});
load(false);
})();
