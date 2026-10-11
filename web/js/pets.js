/* 参数化小伙伴：按名字哈希确定性地组合 耳朵 × 配色 × 配件 × 眼镜。
   形状语言参照主页的幽灵猫（3–4px 描边、方身波浪底、腮红、摇尾巴）。 */
(function(){
var INK="#3b2f4a",PINK="#ff9db5",HOT="#ff6f91",PAPER="#fffdf9",SUN="#ffd36e",SKY="#9fe3ff",PUR="#7c6bd6",LEAF="#7fd8bd";
var WAVE=" L142 160 Q131 150 121 162 Q110 150 100 162 Q90 150 79 162 Q69 150 58 160Z";
var BODY={cat:"M58 160 L58 92 L66 58 L84 76 Q100 70 116 76 L134 58 L142 92"+WAVE,round:"M58 160 L58 106 Q58 70 100 70 Q142 70 142 106"+WAVE};
var EARS=["cat","bunny","bear","mouse","fox","sprout"];
var EAR_ZH={cat:"猫耳",bunny:"兔耳",bear:"熊耳",mouse:"鼠耳",fox:"狐耳",sprout:"小芽"};
var COLORS={mint:"#cdf3e8",blue:"#d4ecff",yellow:"#fff0bf",pink:"#ffd6e2",lav:"#e9e3ff",peach:"#ffe0c9"};
var COLOR_KEYS=["mint","blue","yellow","pink","lav","peach"];
var COLOR_ZH={mint:"薄荷",blue:"天蓝",yellow:"奶黄",pink:"樱粉",lav:"淡紫",peach:"蜜桃",white:"奶白"};
var TOOLS=["cube","paper","wrench","pencil","mug","lens","none"];
var TOOL_ZH={cube:"方块",paper:"日志单",wrench:"扳手",pencil:"铅笔",mug:"马克杯",lens:"放大镜",none:"空手"};
function star(x,y,c,cls){return '<g transform="translate('+x+','+y+')"><path class="'+(cls||"")+'" d="M0 -9 l3 7 7 1 -5 5 1 7 -6 -4 -6 4 1 -7 -5 -5 7 -1z" fill="'+c+'" stroke="'+INK+'" stroke-width="3" stroke-linejoin="round"/></g>';}
function heart(x,y,cls){return '<g transform="translate('+x+','+y+')"><path class="'+cls+'" d="M0 6 C-12 -2 -9 -12 -3 -11 C-1 -11 0 -9 0 -8 C0 -9 1 -11 3 -11 C9 -12 12 -2 0 6Z" fill="'+HOT+'" stroke="'+INK+'" stroke-width="3" stroke-linejoin="round"/></g>';}
function face(m){var e="";
 if(m==="idle")e='<path class="k" d="M77 110 Q84 117 91 110" fill="none"/><path class="k" d="M109 110 Q116 117 123 110" fill="none"/><path class="k" d="M96 127 Q100 130 104 127" fill="none" stroke-width="3.5"/>';
 else if(m==="happy")e='<path class="k" d="M77 114 Q84 103 91 114" fill="none"/><path class="k" d="M109 114 Q116 103 123 114" fill="none"/><path class="k" d="M90 121 Q100 139 110 121Z" fill="'+HOT+'" stroke-width="3.5"/>';
 else if(m==="ask")e='<g class="eye"><ellipse cx="84" cy="109" rx="7" ry="9" fill="'+INK+'"/><circle cx="86.5" cy="105.5" r="2.4" fill="#fff"/></g><g class="eye"><ellipse cx="116" cy="109" rx="7" ry="9" fill="'+INK+'"/><circle cx="118.5" cy="105.5" r="2.4" fill="#fff"/></g><ellipse class="k" cx="100" cy="128" rx="4.5" ry="5" fill="'+HOT+'" stroke-width="3"/>';
 else if(m==="working")e='<g class="eye"><ellipse cx="80" cy="112" rx="5.5" ry="7" fill="'+INK+'"/></g><g class="eye"><ellipse cx="112" cy="112" rx="5.5" ry="7" fill="'+INK+'"/></g><path class="k" d="M88 126 Q92 131 96 126 Q100 131 104 126" fill="none" stroke-width="3.5"/>';
 else e='<g class="eye"><ellipse cx="84" cy="110" rx="6" ry="8" fill="'+INK+'"/></g><g class="eye"><ellipse cx="116" cy="110" rx="6" ry="8" fill="'+INK+'"/></g><path class="k" d="M92 124 Q96 130 100 124 Q104 130 108 124" fill="none" stroke-width="3.5"/>';
 return e+'<circle cx="72" cy="126" r="6.5" fill="'+PINK+'" opacity=".85"/><circle cx="128" cy="126" r="6.5" fill="'+PINK+'" opacity=".85"/>';}
function tool(t){
 if(t==="cube")return '<g class="tool"><path class="k" d="M44 112 L64 122 L44 132 L24 122Z" fill="'+SUN+'" stroke-width="3.5"/><path class="k" d="M24 122 L44 132 L44 154 L24 144Z" fill="'+PAPER+'" stroke-width="3.5"/><path class="k" d="M44 132 L64 122 L64 144 L44 154Z" fill="'+SKY+'" stroke-width="3.5"/><path d="M34 117 L54 127" stroke="'+INK+'" stroke-width="2.5" stroke-linecap="round" opacity=".5"/></g>';
 if(t==="paper")return '<g class="tool"><rect class="k" x="18" y="108" width="34" height="44" rx="6" fill="'+PAPER+'" stroke-width="3.5" transform="rotate(-10 35 130)"/><path d="M26 121 L44 118 M27 130 L45 127 M28 139 L39 137" stroke="'+PUR+'" stroke-width="3" stroke-linecap="round" fill="none"/></g>';
 if(t==="wrench")return '<g class="tool"><path class="k" d="M30 152 L44 120" stroke-width="10"/><path d="M30 152 L44 120" stroke="'+SKY+'" stroke-width="3.5" stroke-linecap="round"/><path class="k" d="M38 108 a12 12 0 1 0 16 6 l-7 3 -4 -6 3 -7 a12 12 0 0 0 -8 4Z" fill="'+SKY+'" stroke-width="3.5"/></g>';
 return "";}
function toolX(t){
 if(t==="pencil")return '<g class="tool"><path class="k" d="M26 154 L50 112" stroke-width="12"/><path d="M26 154 L50 112" stroke="'+SUN+'" stroke-width="5" stroke-linecap="round"/><path class="k" d="M44 108 L56 102 L54 116Z" fill="'+HOT+'" stroke-width="3"/><path class="k" d="M22 160 L24 150 L31 154Z" fill="'+INK+'" stroke-width="2.5"/></g>';
 if(t==="mug")return '<g class="tool"><path class="k" d="M20 124 L52 124 L50 152 Q49 156 45 156 L27 156 Q23 156 22 152Z" fill="'+PAPER+'" stroke-width="3.5"/><path class="k" d="M51 130 Q62 131 60 141 Q58 148 50 147" fill="none" stroke-width="3.5"/><rect x="24" y="134" width="24" height="6" fill="'+HOT+'"/><path d="M30 116 Q26 110 30 104 M40 116 Q36 110 40 104" stroke="'+INK+'" stroke-width="3" stroke-linecap="round" fill="none" opacity=".55"/></g>';
 if(t==="lens")return '<g class="tool"><path class="k" d="M44 136 L26 158" stroke-width="9"/><circle class="k" cx="50" cy="124" r="15" fill="'+SKY+'" stroke-width="4"/><path d="M42 118 Q46 113 52 114" stroke="#fff" stroke-width="3.5" stroke-linecap="round" fill="none"/></g>';
 if(t==="none")return "";
 return tool(t);}
function extras(m,id){
 if(m==="ask")return '<g class="qm"><rect class="k" x="140" y="20" width="40" height="38" rx="13" fill="'+PAPER+'" stroke-width="3.5"/><path class="k" d="M148 56 L146 68 L158 57" fill="'+PAPER+'" stroke-width="3.5"/><path d="M146 55 L159 55" stroke="'+PAPER+'" stroke-width="5"/><text x="160" y="50" text-anchor="middle" font-family="Fredoka" font-weight="700" font-size="28" fill="'+HOT+'">?</text></g>';
 if(m==="idle")return '<text class="zz" x="142" y="70" font-family="Fredoka" font-weight="700" font-size="24" fill="'+INK+'">z</text><text class="zz z2" x="158" y="46" font-family="Fredoka" font-weight="700" font-size="32" fill="'+INK+'">z</text>';
 if(m==="happy")return heart(40,58,"ht")+heart(162,46,"ht h2")+star(170,96,SUN,"ht")+star(30,96,SKY,"ht h2");
 if(m==="working")return star(22,96,SUN,"spark");
 return "";}
function hash(str){var h=2166136261;for(var i=0;i<str.length;i++){h^=str.charCodeAt(i);h=Math.imul(h,16777619);}
 h^=h>>>16;h=Math.imul(h,2246822507);h^=h>>>13;h=Math.imul(h,3266489909);h^=h>>>16;return h>>>0;}
function gen(name,salt){var h=hash(String(name||"")+(salt?"#"+salt:""));
 return {ear:EARS[h%6],color:COLOR_KEYS[Math.floor(h/6)%6],tool:TOOLS[Math.floor(h/36)%7],glasses:Math.floor(h/252)%10<3};}
function draw(L,m,o){o=o||{};m=m||"normal";var fill=COLORS[L.color]||(L.color==="white"?"#fffaf4":COLORS.lav);
 var ear=L.ear||"cat",shp=(ear==="cat")?BODY.cat:BODY.round,back="",tail="",front="";
 if(ear==="bunny"){back='<g class="ear"><ellipse class="k" cx="80" cy="46" rx="12" ry="31" fill="'+fill+'" transform="rotate(-10 80 46)"/><ellipse cx="80" cy="50" rx="5" ry="19" fill="'+PINK+'" transform="rotate(-10 80 50)"/></g><g class="ear e2"><ellipse class="k" cx="120" cy="46" rx="12" ry="31" fill="'+fill+'" transform="rotate(10 120 46)"/><ellipse cx="120" cy="50" rx="5" ry="19" fill="'+PINK+'" transform="rotate(10 120 50)"/></g>';tail='<circle class="k tail" cx="146" cy="140" r="9" fill="'+PAPER+'" stroke-width="3.5"/>';}
 else if(ear==="bear"){back='<g class="ear"><circle class="k" cx="68" cy="80" r="16" fill="'+fill+'"/><circle cx="68" cy="80" r="7" fill="'+PINK+'"/></g><g class="ear e2"><circle class="k" cx="132" cy="80" r="16" fill="'+fill+'"/><circle cx="132" cy="80" r="7" fill="'+PINK+'"/></g>';tail='<path class="k tail" d="M140 146 Q160 146 158 128" fill="none"/>';}
 else if(ear==="mouse"){back='<g class="ear"><circle class="k" cx="62" cy="74" r="23" fill="'+fill+'"/><circle cx="62" cy="74" r="12" fill="'+PINK+'"/></g><g class="ear e2"><circle class="k" cx="138" cy="74" r="23" fill="'+fill+'"/><circle cx="138" cy="74" r="12" fill="'+PINK+'"/></g>';tail='<path class="k tail" d="M140 148 Q168 150 166 126 Q164 112 176 108" fill="none" stroke-width="3.5"/>';}
 else if(ear==="fox"){back='<g class="ear"><path class="k" d="M62 92 L58 42 L96 74Z" fill="'+fill+'"/><path d="M66 82 L64 56 L84 74Z" fill="'+PINK+'"/></g><g class="ear e2"><path class="k" d="M138 92 L142 42 L104 74Z" fill="'+fill+'"/><path d="M134 82 L136 56 L116 74Z" fill="'+PINK+'"/></g>';tail='<path class="k tail" d="M140 150 Q178 150 172 112 Q166 128 146 132" fill="'+fill+'"/>';}
 else if(ear==="sprout"){back='<g class="ear"><path class="k" d="M100 72 Q98 56 104 44" fill="none" stroke-width="3.5"/><path class="k" d="M103 50 Q90 30 72 40 Q86 56 103 50Z" fill="'+LEAF+'" stroke-width="3.5"/><path class="k" d="M104 46 Q118 26 134 36 Q122 52 104 46Z" fill="'+LEAF+'" stroke-width="3.5"/></g>';tail='<path class="k tail" d="M140 140 Q166 136 162 114" fill="none"/>';}
 else tail='<path class="k tail" d="M140 140 Q172 132 166 104" fill="none"/>';
 if(L.glasses&&!L.outfit){var w=m==="working"?-4:0;front+='<g fill="none" stroke="'+INK+'" stroke-width="3"><circle cx="'+(84+w)+'" cy="111" r="13"/><circle cx="'+(116+w)+'" cy="111" r="13"/><path d="M'+(97+w)+' 109 Q'+(100+w)+' 105 '+(103+w)+' 109"/></g>';}
 if(L.outfit==="thief")front+='<path class="k" fill-rule="evenodd" fill="#2e2440" d="M58 100 Q66 92 84 96 Q100 101 116 96 Q134 92 142 100 L153 91 Q151 113 136 119 Q118 124 100 116 Q82 124 64 119 Q49 113 47 91 Z M74 109 a10 8.5 0 1 0 20 0 a10 8.5 0 1 0 -20 0 Z M106 109 a10 8.5 0 1 0 20 0 a10 8.5 0 1 0 -20 0 Z" stroke-width="3"/><path class="k" fill="#2e2440" d="M59 136 Q78 142 91 140 L100 155 L109 140 Q122 142 141 136 L142 160 Q131 150 121 162 Q110 150 100 162 Q90 150 79 162 Q69 150 58 160 Z"/><path class="k" stroke-width="3" fill="'+HOT+'" d="M100 142 l-10 -6 0 12z M100 142 l10 -6 0 12z"/><g><rect class="k" x="86" y="45" width="28" height="25" rx="3" fill="#2e2440"/><rect x="88" y="59" width="24" height="5" fill="'+HOT+'"/><rect class="k" x="76" y="67" width="48" height="7" rx="3.5" fill="#2e2440"/></g>';
 return '<svg class="pet '+m+'" viewBox="0 0 200 200" aria-hidden="true">'+(o.noShadow?"":'<ellipse class="sh" cx="100" cy="188" rx="42" ry="7" fill="'+INK+'" opacity=".18"/>')+
  '<g class="fl">'+back+tail+'<path class="k" d="'+shp+'" fill="'+fill+'"/>'+face(m)+front+toolX(L.tool||"none")+'</g>'+extras(m)+'</svg>';}
function describe(L){return [EAR_ZH[L.ear],COLOR_ZH[L.color],TOOL_ZH[L.tool||"none"]].concat(L.glasses?["眼镜"]:[]).join(" · ");}
window.PETS={gen:gen,draw:draw,describe:describe,hash:hash,COLORS:COLORS,COLOR_KEYS:COLOR_KEYS,COLOR_ZH:COLOR_ZH,EARS:EARS,TOOLS:TOOLS};
window.PET=function(look,m,o){return draw(look,m,o);};
window.PETHEAD=function(look,m){return draw(look,m,{noShadow:true});};
window.STAR=star;window.HEART=heart;
})();
