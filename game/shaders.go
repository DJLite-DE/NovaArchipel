package game

const glslCommon = `
float hash12(vec2 p){ vec3 p3 = fract(vec3(p.xyx) * 0.1031); p3 += dot(p3, p3.yzx + 33.33); return fract((p3.x + p3.y) * p3.z); }
float vnoise(vec2 p){ vec2 i=floor(p), f=fract(p); vec2 u=f*f*(3.0-2.0*f);
  return mix(mix(hash12(i),hash12(i+vec2(1,0)),u.x), mix(hash12(i+vec2(0,1)),hash12(i+vec2(1,1)),u.x), u.y); }
float fbm(vec2 p){ float v=0.0, a=0.5; for(int i=0;i<4;i++){ v+=a*vnoise(p); p*=2.03; a*=0.5; } return v; }
`

const vsMesh = `#version 330 core
layout(location=0) in vec3 aPos;
layout(location=1) in vec3 aNrm;
layout(location=2) in vec4 aCol;
layout(location=3) in vec4 iPos;
layout(location=4) in vec4 iScl;
uniform mat4 uVP;
uniform mat4 uModel;
uniform int uInst;
uniform float uTime;
out vec3 vPos; out vec3 vNrm; out vec4 vCol; out float vH;
void main(){
  vec3 p = aPos; vec3 n = aNrm; vec4 col = aCol; vec3 wp;
  if(uInst==1){
    float c=cos(iPos.w), s=sin(iPos.w);
    vec3 q = p*iScl.x;
    if(col.a>3.5 && col.a<4.5){ float sw = sin(uTime*1.4+iPos.x*0.9+iPos.z*0.6)*0.035*q.y; q.x+=sw; q.z+=sw*0.7; }
    if(col.a>10.5 && col.a<11.5){ q.y += sin(uTime*11.0 + iPos.x*5.0 + iPos.z*3.0)*abs(q.z)*0.8; col.a = 0.0; }
    wp = vec3(c*q.x + s*q.z, q.y, -s*q.x + c*q.z) + iPos.xyz;
    n = vec3(c*n.x + s*n.z, n.y, -s*n.x + c*n.z);
    col.rgb *= iScl.yzw;
    vH = p.y;
  } else {
    wp = (uModel*vec4(p,1.0)).xyz;
    n = mat3(uModel)*n;
    vH = p.y;
  }
  vPos=wp; vNrm=n; vCol=col;
  gl_Position = uVP*vec4(wp,1.0);
}`

const fsMesh = `#version 330 core
in vec3 vPos; in vec3 vNrm; in vec4 vCol; in float vH;
uniform vec3 uSun; uniform vec3 uSunCol; uniform vec3 uSky; uniform vec3 uGnd; uniform vec3 uCam;
uniform sampler2DShadow uShadow; uniform mat4 uLightVP;
uniform vec4 uTint; uniform float uAlpha; uniform float uTime; uniform vec3 uFog; uniform float uFogD;
uniform float uAO; uniform float uNight; uniform int uPCF; uniform float uClouds;
uniform int uClip; uniform float uHFog; uniform float uMoon;
out vec4 frag;
` + glslCommon + `
float shadowAt(vec3 p, vec3 n){
  vec4 lp = uLightVP*vec4(p + n*0.04, 1.0);
  vec3 c = lp.xyz/lp.w*0.5+0.5;
  if(c.x<0.0||c.x>1.0||c.y<0.0||c.y>1.0||c.z>1.0) return 1.0;
  vec2 ts = 1.0/vec2(textureSize(uShadow,0));
  float s=0.0; float cnt=0.0;
  for(int x=-uPCF;x<=uPCF;x++) for(int y=-uPCF;y<=uPCF;y++){ s += texture(uShadow, vec3(c.xy+vec2(x,y)*ts*1.1, c.z-0.0010)); cnt+=1.0; }
  return s/cnt;
}
float cloudAt(vec3 p){
  return uClouds * smoothstep(0.52, 0.8, fbm(p.xz*0.022 + vec2(uTime*0.012, uTime*0.007)));
}
vec3 skyRefl(vec3 r){ float t = clamp(r.y*0.5+0.5, 0.0, 1.0); return mix(uGnd*0.8, uSky*1.25, t); }
float ggx(vec3 N, vec3 V, vec3 L, float rough, float f0){
  vec3 H = normalize(V+L);
  float a = rough*rough; float a2 = a*a;
  float nh = max(dot(N,H),0.0), nl = max(dot(N,L),0.0), nv = max(dot(N,V),0.001), vh = max(dot(V,H),0.0);
  float d = nh*nh*(a2-1.0)+1.0; float D = a2/(3.14159*d*d);
  float F = f0 + (1.0-f0)*pow(1.0-vh, 5.0);
  float k = (rough+1.0)*(rough+1.0)/8.0;
  float G = (nl/(nl*(1.0-k)+k))*(nv/(nv*(1.0-k)+k));
  return D*F*G/max(4.0*nl*nv, 0.001)*nl;
}
void main(){
  if(uClip==1 && vPos.y < -0.02) discard;
  vec3 N = normalize(vNrm);
  if(!gl_FrontFacing) N = -N;
  int m = int(vCol.a+0.5);
  vec3 alb = vCol.rgb;
  vec3 V = normalize(uCam - vPos);
  float sh = shadowAt(vPos, N) * (1.0 - 0.45*cloudAt(vPos));
  float ndl = max(dot(N,uSun),0.0);
  float specK = 0.08, shin = 24.0;
  vec3 emis = vec3(0.0);
  vec3 extra = vec3(0.0);
  if(m==6){
    float n1 = fbm(vPos.xz*0.35), n2 = fbm(vPos.xz*3.0+7.0), n3 = vnoise(vPos.xz*14.0);
    alb *= 0.72 + 0.45*n1 + 0.18*n2 + 0.08*n3;
    // sattere, dunklere Grasflecken
    alb = mix(alb, alb*vec3(0.75,0.95,0.7), smoothstep(0.45,0.7,n1)*step(0.25,vPos.y)*(1.0-uMoon));
    alb *= 1.0 - 0.38*uMoon;
    float slope = 1.0 - N.y;
    alb = mix(alb, vec3(0.20,0.19,0.18)*(0.8+0.4*n2), smoothstep(0.28,0.55,slope));
    specK = 0.03;
  } else if(m==1){ specK = 0.45; shin = 70.0; }
  else if(m==7){ specK = 0.9; shin = 90.0; }
  else if(m==2){
    vec2 uv = abs(N.x)>0.5 ? vec2(vPos.z, vPos.y) : (abs(N.z)>0.5 ? vec2(vPos.x, vPos.y) : vPos.xz);
    vec2 g = fract(uv*vec2(5.0, 6.5));
    vec2 cell = floor(uv*vec2(5.0,6.5));
    float glass = step(0.07,g.x)*step(0.12,g.y);
    vec3 R = reflect(-V,N);
    float fres = 0.18 + 0.82*pow(1.0-max(dot(N,V),0.0), 4.0);
    vec3 frameC = vec3(0.82,0.85,0.88);
    vec3 glassC = vec3(0.03,0.07,0.11);
    alb = mix(frameC, glassC, glass);
    extra = skyRefl(R)*fres*glass*0.9;
    float lit = step(0.7, hash12(cell+floor(vPos.xz*0.37)*13.0));
    float tintv = hash12(cell*1.7+3.0);
    alb = mix(alb, alb*vec3(0.85,0.95,1.15), glass*tintv*0.6);
    emis = glass*lit*mix(vec3(1.0,0.72,0.42), vec3(0.75,0.85,1.0), step(0.8, tintv))*(uNight*0.75);
    specK = mix(0.3, 1.5, glass); shin = 180.0;
  }
  else if(m==3){ emis = alb*(2.2+uNight*2.0); }
  else if(m==9){ emis = alb*(0.12+uNight*7.0); }
  else if(m==10){
    vec2 q = vec2(vPos.x + vPos.z, vPos.y);
    float band = floor(q.y*10.0);
    float bars = step(0.45, fract(q.x*9.0 + band*0.37 - uTime*0.35 + hash12(vec2(band,3.0))));
    vec3 pal = 0.5 + 0.5*cos(6.2831*(vec3(0.0,0.33,0.67) + fract(uTime*0.05 + q.y*0.4 + q.x*0.2)));
    emis = mix(pal*0.35, pal, bars) * (0.9 + uNight*2.5);
    alb = vec3(0.02); specK = 0.6; shin = 120.0;
  }
  else if(m==4){ specK=0.05; alb *= 0.9+0.2*vnoise(vPos.xz*3.0); }
  else if(m==5){
    vec2 uv = vPos.xz*6.0;
    float grid = step(0.08,fract(uv.x))*step(0.08,fract(uv.y));
    alb = mix(vec3(0.6,0.65,0.7), vec3(0.02,0.05,0.14), grid);
    extra = skyRefl(reflect(-V,N))*0.25*grid;
    specK = 1.2; shin = 120.0;
  }
  else if(m==8){
    vec3 R = reflect(-V,N);
    alb = vec3(0.05,0.45,0.55);
    extra = skyRefl(R)*0.35;
    specK = 1.4; shin = 200.0;
    emis = vec3(0.02,0.12,0.14)*(1.0+uNight*3.0);
  }
  vec3 amb = mix(uGnd*1.2, uSky*0.9 + vec3(0.06,0.05,0.03), N.y*0.5+0.5) * alb;
  float ao = 1.0;
  if(uAO>0.5){ ao = clamp(0.55 + vH*0.9, 0.55, 1.0); }
  float rough = clamp(1.0 - log2(shin)/8.0, 0.06, 0.95);
  float f0 = (m==7) ? 0.55 : 0.04;
  float sp = ggx(N, V, uSun, rough, f0) * specK * 2.0;
  // Umgebungsreflexion auf glatten Flächen
  if(m==1 || m==7){
    float fr = f0 + (1.0-f0)*pow(1.0-max(dot(N,V),0.0), 5.0);
    extra += skyRefl(reflect(-V,N)) * fr * (m==7 ? 0.6 : 0.12) * (1.0-rough);
  }
  // Kaustiken auf dem Meeresgrund
  if(m==6 && vPos.y < 0.0){
    vec2 cp = vPos.xz*3.0;
    float c1 = vnoise(cp + vec2(uTime*0.6, uTime*0.4));
    float c2 = vnoise(cp*1.3 - vec2(uTime*0.5, -uTime*0.3));
    float ca = pow(1.0 - abs(c1-c2), 8.0);
    emis += uSunCol * ca * 0.12 * smoothstep(-1.6, -0.05, vPos.y) * sh;
    alb *= mix(vec3(0.75,0.9,0.95), vec3(1.0), smoothstep(-1.5, 0.0, vPos.y));
  }
  vec3 col = amb*ao*0.42 + alb*uSunCol*ndl*sh + uSunCol*sp*sh + extra + emis;
  float d = length(uCam - vPos);
  float f = 1.0 - exp(-d*uFogD);
  f += uHFog * exp(-max(vPos.y, 0.0)*1.6) * (1.0 - exp(-d*0.012)) * 0.22;
  col = mix(col, uFog, clamp(f,0.0,0.9));
  col = mix(col, uTint.rgb, uTint.a);
  frag = vec4(col, uAlpha);
}`

const fsDepth = `#version 330 core
void main(){}`

const vsWater = `#version 330 core
layout(location=0) in vec3 aPos;
uniform mat4 uVP; uniform float uTime;
out vec3 vPos;
void main(){
  vec3 p = aPos;
  p.y = 0.0 + sin(p.x*0.9+uTime*1.2)*0.015 + sin(p.z*1.1+uTime*0.9)*0.015;
  vPos = p;
  gl_Position = uVP*vec4(p,1.0);
}`

const fsWater = `#version 330 core
in vec3 vPos;
uniform vec3 uSun; uniform vec3 uSunCol; uniform vec3 uSky; uniform vec3 uGnd; uniform vec3 uCam;
uniform float uTime; uniform sampler2D uHeight; uniform float uMapN;
uniform sampler2DShadow uShadow; uniform mat4 uLightVP; uniform vec3 uFog; uniform float uFogD; uniform float uNight;
uniform float uClouds; uniform float uStorm;
uniform sampler2D uRefl; uniform int uReflOn; uniform vec2 uView; uniform float uHFog; uniform float uMoon;
out vec4 frag;
` + glslCommon + `
float shadowAt(vec3 p){
  vec4 lp = uLightVP*vec4(p, 1.0);
  vec3 c = lp.xyz/lp.w*0.5+0.5;
  if(c.x<0.0||c.x>1.0||c.y<0.0||c.y>1.0||c.z>1.0) return 1.0;
  return texture(uShadow, vec3(c.xy, c.z-0.002));
}
float waveH(vec2 p){
  float t = uTime;
  float h = 0.0;
  h += sin(dot(p, vec2(0.8,0.6))*2.1 + t*1.3)*0.5;
  h += sin(dot(p, vec2(-0.5,0.9))*3.3 + t*1.7)*0.3;
  h += sin(dot(p, vec2(0.95,-0.3))*5.7 + t*2.3)*0.15;
  h += (vnoise(p*3.0 + vec2(t*0.35, t*0.2))-0.5)*0.5;
  h += (vnoise(p*7.0 - vec2(t*0.5, -t*0.3))-0.5)*0.25;
  return h*(0.035 + uStorm*0.05);
}
void main(){
  vec2 p = vPos.xz;
  if(uMoon > 0.5){
    // Staubmeer des Mondes
    float sh = shadowAt(vPos);
    float n1 = vnoise(p*0.35), n2 = vnoise(p*2.2), n3 = vnoise(p*9.0);
    vec3 c = vec3(0.30,0.295,0.29)*(0.72 + 0.4*n1)*(0.88 + 0.16*n2)*(0.94+0.08*n3);
    // kleine Krater
    vec2 cc = floor(p*0.25);
    vec2 cp = fract(p*0.25) - 0.5 - (vec2(hash12(cc), hash12(cc+5.3)) - 0.5)*0.5;
    float cr = length(cp);
    float rim = smoothstep(0.22, 0.18, cr) - smoothstep(0.18, 0.1, cr)*1.6;
    c *= 1.0 + rim*0.22*step(0.55, hash12(cc+9.1));
    vec3 col = c*(0.06 + uSunCol*0.42*max(uSun.y, 0.0)*sh);
    float d = length(uCam - vPos);
    col = mix(col, uFog, clamp(1.0 - exp(-d*uFogD), 0.0, 0.9));
    frag = vec4(col, 1.0);
    return;
  }
  float e = 0.05;
  float h0 = waveH(p);
  vec3 N = normalize(vec3(-(waveH(p+vec2(e,0))-h0)/e, 1.0, -(waveH(p+vec2(0,e))-h0)/e));
  vec2 uv = (p+0.5)/uMapN;
  float ground = -2.2;
  if(uv.x>=0.0 && uv.y>=0.0 && uv.x<=1.0 && uv.y<=1.0) ground = texture(uHeight, uv).r;
  float depth = max(0.0, -ground);
  vec3 V = normalize(uCam - vPos);
  vec3 deep = vec3(0.002,0.03,0.09);
  vec3 mid  = vec3(0.004,0.10,0.22);
  vec3 shal = vec3(0.03,0.42,0.48);
  vec3 wc = mix(shal, mid, smoothstep(0.05,0.7,depth));
  wc = mix(wc, deep, smoothstep(0.7,2.0,depth));
  float sh = shadowAt(vPos) * (1.0 - 0.45*uClouds*smoothstep(0.52, 0.8, fbm(vPos.xz*0.022 + vec2(uTime*0.012, uTime*0.007))));
  vec3 R = reflect(-V, N);
  float fres = 0.04 + 0.96*pow(1.0-max(dot(N,V),0.0), 5.0);
  vec3 sky = mix(uGnd*0.6, uSky*1.1, clamp(R.y,0.0,1.0));
  float rw = 0.55;
  if(uReflOn==1){
    vec2 ruv = gl_FragCoord.xy/uView + N.xz*0.035;
    sky = texture(uRefl, clamp(ruv, 0.001, 0.999)).rgb;
    rw = 1.0;
  }
  vec3 H = normalize(uSun+V);
  float spec = pow(max(dot(N,H),0.0), 400.0)*6.0 + pow(max(dot(N,H),0.0), 60.0)*0.25;
  fres = max(fres, 0.12*rw);
  vec3 col = wc*(0.45+0.55*sh)*(0.6+0.6*max(dot(N,uSun),0.0))*(1.0-fres*0.6) + sky*fres*rw*0.9 + uSunCol*spec*sh;
  // Gischt an der Küste
  float foamN = vnoise(p*6.0 + vec2(uTime*0.6, uTime*0.4));
  float foam = smoothstep(0.16+uStorm*0.25, 0.0, depth) * smoothstep(0.35, 0.75, foamN + 0.35*sin(depth*40.0 - uTime*2.5));
  foam += uStorm*smoothstep(0.72, 0.9, vnoise(p*2.5 + vec2(uTime*0.8, uTime*0.3)))*0.6;
  col = mix(col, vec3(0.95,0.98,1.0)*(0.6+0.4*sh), foam*0.85);
  col += vec3(0.0,0.03,0.05)*uNight;
  float a = clamp(0.45 + depth*1.6, 0.45, 0.98);
  a = max(a, foam);
  float d = length(uCam - vPos);
  float f = 1.0 - exp(-d*uFogD) + uHFog*(1.0 - exp(-d*0.012))*0.22;
  col = mix(col, uFog, clamp(f,0.0,0.9));
  frag = vec4(col, a);
}`

const vsQuad = `#version 330 core
out vec2 vUV;
void main(){
  vec2 p = vec2((gl_VertexID<<1)&2, gl_VertexID&2);
  vUV = p;
  gl_Position = vec4(p*2.0-1.0, 0.0, 1.0);
}`

const fsSky = `#version 330 core
in vec2 vUV; uniform mat4 uInvVP; uniform vec3 uSun; uniform vec3 uSky; uniform vec3 uHorizon; uniform vec3 uSunCol;
uniform vec3 uRealSun; uniform float uNight; uniform float uTime; uniform float uMoon;
out vec4 frag;
` + glslCommon + `
void main(){
  vec4 a = uInvVP*vec4(vUV*2.0-1.0, -1.0, 1.0); vec4 b = uInvVP*vec4(vUV*2.0-1.0, 1.0, 1.0);
  vec3 dir = normalize(b.xyz/b.w - a.xyz/a.w);
  float t = clamp(dir.y, 0.0, 1.0);
  vec3 col = mix(uHorizon, uSky, pow(t, 0.55));
  float s = max(dot(dir, uRealSun), 0.0);
  col += uSunCol*(pow(s, 900.0)*10.0 + pow(s, 10.0)*0.3)*(1.0-uNight);
  // Sterne und Mond
  vec2 sp = vec2(atan(dir.z, dir.x)*80.0, dir.y*160.0);
  float st = step(0.985, hash12(floor(sp))) * hash12(floor(sp)+3.1);
  col += vec3(0.9,0.95,1.0)*st*uNight*t*1.5;
  float m = max(dot(dir, -uRealSun), 0.0);
  col += vec3(0.8,0.85,1.0)*(pow(m, 1800.0)*4.0 + pow(m, 30.0)*0.08)*uNight;
  if(uMoon > 0.5){
    // Sterne immer sichtbar, dazu die Erde am Himmel
    col += vec3(0.9,0.95,1.0)*st*1.4*smoothstep(0.0,0.15,dir.y);
    float st2 = step(0.995, hash12(floor(sp*2.3)+7.0));
    col += vec3(1.0,0.9,0.8)*st2*0.6*smoothstep(0.0,0.15,dir.y);
    vec3 ed = normalize(vec3(-0.55, 0.42, -0.72));
    float e = dot(dir, ed);
    float R = 0.075;
    float cr = cos(R);
    vec3 ax = normalize(cross(ed, vec3(0.0,1.0,0.0)));
    vec3 ay = cross(ax, ed);
    vec2 q = vec2(dot(dir, ax), dot(dir, ay))/sin(R);
    float r2 = dot(q,q);
    if(e > cr){
      vec3 n = vec3(q, sqrt(max(0.0, 1.0-r2)));
      float land = smoothstep(0.52, 0.6, fbm(q*3.0 + vec2(uTime*0.004, 0.0)));
      float cl = smoothstep(0.55, 0.75, fbm(q*5.0 + vec2(uTime*0.01, 3.0)));
      vec3 ec = mix(vec3(0.05,0.2,0.6), vec3(0.25,0.45,0.2), land);
      ec = mix(ec, vec3(0.95), cl*0.85);
      float lit = clamp(dot(n, normalize(vec3(0.7,0.25,0.65))), 0.0, 1.0);
      vec3 night = vec3(1.0,0.75,0.4)*land*smoothstep(0.3,0.0,lit)*step(0.6, hash12(floor(q*60.0)))*0.5;
      col = ec*(0.04 + lit*1.4) + night + vec3(0.3,0.5,1.0)*pow(1.0-n.z, 3.0)*lit*0.8;
    } else {
      col += vec3(0.25,0.45,1.0)*exp(-(sqrt(r2)-1.0)*18.0)*0.25*step(1.0, r2);
    }
  }
  if(dir.y < 0.0) col = uHorizon*0.9;
  frag = vec4(col, 1.0);
}`

const fsDown = `#version 330 core
in vec2 vUV; uniform sampler2D uTex; uniform vec2 uTexel;
out vec4 frag;
vec3 smp(vec2 o){
  vec3 c = texture(uTex, vUV + uTexel*o).rgb;
  if(any(isnan(c)) || any(isinf(c))) return vec3(0.0);
  return clamp(c, vec3(0.0), vec3(64.0));
}
void main(){
  vec3 c = smp(vec2(-1,-1)) + smp(vec2(1,-1)) + smp(vec2(-1,1)) + smp(vec2(1,1));
  frag = vec4(c*0.25, 1.0);
}`

const fsBlur = `#version 330 core
in vec2 vUV; uniform sampler2D uTex; uniform vec2 uDir;
out vec4 frag;
void main(){
  float w[5] = float[](0.227027, 0.1945946, 0.1216216, 0.054054, 0.016216);
  vec3 c = texture(uTex, vUV).rgb*w[0];
  for(int i=1;i<5;i++){ c += texture(uTex, vUV+uDir*float(i)*1.5).rgb*w[i]; c += texture(uTex, vUV-uDir*float(i)*1.5).rgb*w[i]; }
  frag = vec4(c,1.0);
}`

const fsComposite = `#version 330 core
in vec2 vUV;
uniform sampler2D uScene; uniform sampler2D uBlur; uniform sampler2D uUI;
uniform vec4 uRects[40]; uniform int uNRect; uniform vec2 uScreen; uniform float uBloom; uniform int uUIOn;
uniform float uRain; uniform float uFlash; uniform float uTime;
uniform sampler2D uDepth; uniform int uDOF; uniform float uFocus; uniform vec2 uNF; uniform float uGrain;
uniform vec2 uSunUV; uniform float uRays; uniform vec3 uRayCol;
out vec4 frag;
float h12(vec2 p){ vec3 p3 = fract(vec3(p.xyx) * 0.1031); p3 += dot(p3, p3.yzx + 33.33); return fract((p3.x + p3.y) * p3.z); }
vec3 aces(vec3 x){ return clamp((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14), 0.0, 1.0); }
float rrect(vec2 p, vec4 r, float rad){
  vec2 c = r.xy + r.zw*0.5; vec2 hs = r.zw*0.5 - rad;
  vec2 q = abs(p-c) - hs;
  return length(max(q,0.0)) + min(max(q.x,q.y),0.0) - rad;
}
void main(){
  vec3 hdr = texture(uScene, vUV).rgb;
  vec3 bl = texture(uBlur, vUV).rgb;
  if(any(isnan(hdr)) || any(isinf(hdr))) hdr = bl;
  if(uDOF==1){
    float dd = texture(uDepth, vUV).r;
    float n = uNF.x, fa = uNF.y;
    float z = (2.0*n*fa)/(fa+n-(dd*2.0-1.0)*(fa-n));
    float amt = clamp(abs(z-uFocus)/(uFocus*0.9) - 0.12, 0.0, 1.0);
    float tilt = smoothstep(0.22, 0.0, vUV.y) + smoothstep(0.82, 1.0, vUV.y);
    amt = clamp(max(amt*0.7, tilt*0.75), 0.0, 1.0);
    hdr = mix(hdr, bl, amt*0.85);
  }
  hdr += max(bl - vec3(0.85), 0.0)*uBloom;
  // Lichtstrahlen: Himmelspixel entlang der Linie zur Sonne aufsummieren
  if(uRays > 0.001){
    vec2 dv = (uSunUV - vUV);
    float acc = 0.0;
    float wsum = 0.0;
    float jit = h12(gl_FragCoord.xy + uTime);
    for(int i=0;i<28;i++){
      float t = (float(i)+jit)/28.0;
      vec2 sp = vUV + dv*t*0.92;
      if(sp.x<0.0||sp.y<0.0||sp.x>1.0||sp.y>1.0) continue;
      float sky = step(0.99995, texture(uDepth, sp).r);
      float wgt = 1.0 - t*0.6;
      acc += sky*wgt;
      wsum += wgt;
    }
    float ray = acc/max(wsum, 0.001);
    float fall = exp(-length(dv*vec2(uScreen.x/uScreen.y, 1.0))*1.6);
    hdr += uRayCol*ray*fall*uRays;
  }
  hdr *= 1.0 + uFlash*2.5;
  vec3 c = aces(hdr*0.95);
  if(uRain > 0.0){
    vec2 q = gl_FragCoord.xy; q.x += q.y*0.18;
    float col = floor(q.x/3.0);
    float sp = 1400.0 + h12(vec2(col,1.0))*600.0;
    float y = fract((q.y + uTime*sp)/ (180.0 + h12(vec2(col,2.0))*220.0));
    float drop = step(0.93, h12(vec2(col, 7.0))) * smoothstep(0.0, 0.08, y) * smoothstep(0.25, 0.08, y);
    c = mix(c, c*vec3(0.78,0.82,0.9), uRain*0.45);
    c += vec3(0.55,0.6,0.7)*drop*uRain*0.35;
  }
  c = pow(c, vec3(1.0/2.2));
  // Farbgebung: leicht kräftiger
  float l = dot(c, vec3(0.299,0.587,0.114));
  c = mix(vec3(l), c, 1.06);
  c = pow(c, vec3(1.06));
  vec2 q = vUV-0.5; c *= 1.0 - dot(q,q)*0.35;
  c += (h12(gl_FragCoord.xy + fract(uTime*7.13)*100.0) - 0.5) * uGrain;
  if(uUIOn==1){
    vec2 px = vec2(gl_FragCoord.x, uScreen.y - gl_FragCoord.y);
    for(int i=0;i<uNRect;i++){
      float d = rrect(px, uRects[i], 10.0);
      if(d < 0.0){
        vec3 g = pow(aces(bl*1.05), vec3(1.0/2.2));
        c = mix(c, g*0.55 + vec3(0.015,0.025,0.04), 0.92);
        break;
      }
    }
    vec4 ui = texture(uUI, vec2(vUV.x, 1.0-vUV.y));
    c = ui.rgb + c*(1.0-ui.a);
  }
  frag = vec4(c, 1.0);
}`

const vsFlat = `#version 330 core
layout(location=0) in vec3 aPos;
uniform mat4 uVP;
void main(){ gl_Position = uVP*vec4(aPos,1.0); }`

const fsFlat = `#version 330 core
uniform vec4 uColor;
out vec4 frag;
void main(){ frag = uColor; }`

const fsSSAO = `#version 330 core
in vec2 vUV; uniform sampler2D uDepth; uniform mat4 uInvProj; uniform mat4 uProj; uniform vec2 uTexel; uniform float uRadius;
out vec4 frag;
vec3 viewPos(vec2 uv){ float d = texture(uDepth, uv).r; vec4 p = uInvProj*vec4(uv*2.0-1.0, d*2.0-1.0, 1.0); return p.xyz/p.w; }
float h12(vec2 p){ vec3 p3 = fract(vec3(p.xyx) * 0.1031); p3 += dot(p3, p3.yzx + 33.33); return fract((p3.x + p3.y) * p3.z); }
void main(){
  float d = texture(uDepth, vUV).r;
  if(d >= 0.9999){ frag = vec4(1.0); return; }
  vec3 P = viewPos(vUV);
  vec3 px = viewPos(vUV + vec2(uTexel.x,0.0)) - P;
  vec3 py = viewPos(vUV + vec2(0.0,uTexel.y)) - P;
  vec3 N = normalize(cross(px, py));
  float ang = h12(gl_FragCoord.xy)*6.2831;
  float occ = 0.0;
  const int K = 16;
  for(int i=0;i<K;i++){
    float fi = float(i);
    float a = ang + fi*2.39996;
    float r = sqrt((fi+0.5)/float(K));
    vec3 dir = vec3(cos(a)*r, sin(a)*r, sqrt(max(0.0,1.0-r*r)));
    vec3 t = normalize(abs(N.y)<0.9 ? cross(N, vec3(0,1,0)) : cross(N, vec3(1,0,0)));
    vec3 bt = cross(N, t);
    vec3 sdir = t*dir.x + bt*dir.y + N*dir.z;
    float sc = uRadius*(0.3 + 0.7*fract(fi*0.618+ang));
    vec3 S = P + sdir*sc;
    vec4 cs = uProj*vec4(S,1.0);
    vec2 suv = cs.xy/cs.w*0.5+0.5;
    if(suv.x<0.0||suv.y<0.0||suv.x>1.0||suv.y>1.0) continue;
    float sd = viewPos(suv).z;
    float range = smoothstep(0.0, 1.0, uRadius/abs(P.z - sd));
    occ += (sd >= S.z + 0.03 ? 1.0 : 0.0) * range;
  }
  float ao = 1.0 - occ/float(K);
  frag = vec4(vec3(pow(ao, 1.6)), 1.0);
}`

const fsAOApply = `#version 330 core
in vec2 vUV; uniform sampler2D uAO; uniform float uStrength;
out vec4 frag;
void main(){ float a = texture(uAO, vUV).r; frag = vec4(vec3(mix(1.0, a, uStrength)), 1.0); }`

const vsGlow = `#version 330 core
layout(location=0) in vec3 aPos;
layout(location=1) in vec3 aNrm;
layout(location=2) in vec4 aCol;
uniform mat4 uVP;
out vec2 vUV; out vec3 vCol;
void main(){ vUV = aNrm.xy; vCol = aCol.rgb; gl_Position = uVP*vec4(aPos,1.0); }`

const fsGlow = `#version 330 core
in vec2 vUV; in vec3 vCol; uniform float uNight;
out vec4 frag;
void main(){ float r = length(vUV); float f = pow(max(0.0, 1.0 - r), 2.2); frag = vec4(vCol*f*uNight*0.85, 1.0); }`

const vsPart = `#version 330 core
layout(location=0) in vec3 aPos;
layout(location=3) in vec4 iPos;
layout(location=4) in vec4 iScl;
uniform mat4 uVP; uniform vec3 uRight; uniform vec3 uUp;
out vec2 vUV; out vec4 vD;
void main(){
  float c = cos(iScl.w), s = sin(iScl.w);
  vec2 q = vec2(c*aPos.x - s*aPos.y, s*aPos.x + c*aPos.y);
  vec3 wp = iPos.xyz + (uRight*q.x + uUp*q.y)*iPos.w;
  vUV = aPos.xy; vD = iScl;
  gl_Position = uVP*vec4(wp,1.0);
}`

const fsPart = `#version 330 core
in vec2 vUV; in vec4 vD;
uniform vec3 uSunCol; uniform vec3 uSky; uniform float uNight; uniform vec3 uFog;
out vec4 frag;
` + glslCommon + `
void main(){
  float r = length(vUV);
  float n = fbm(vUV*1.7 + vD.w*3.0 + vD.z*7.0);
  float a = smoothstep(1.0, 0.15, r + (n-0.5)*0.6) * vD.x;
  if(a < 0.003) discard;
  vec3 base = mix(vec3(0.55,0.55,0.56), vec3(1.0), vD.z);
  float lit = 0.55 + 0.45*clamp(0.6 - vUV.y*0.4 + n*0.3, 0.0, 1.0);
  vec3 col = base * (uSky*0.8 + uSunCol*0.55*lit) * vD.y;
  frag = vec4(col, a);
}`

const vsCloud = `#version 330 core
layout(location=0) in vec3 aPos;
uniform mat4 uVP; uniform float uY;
out vec3 vPos;
void main(){ vec3 p = vec3(aPos.x, uY, aPos.z); vPos = p; gl_Position = uVP*vec4(p,1.0); }`

const fsCloud = `#version 330 core
in vec3 vPos; uniform float uTime; uniform vec3 uSunCol; uniform vec3 uSky; uniform float uFade; uniform float uStorm; uniform vec3 uCam;
out vec4 frag;
` + glslCommon + `
void main(){
  vec2 p = vPos.xz*0.022 + vec2(uTime*0.012, uTime*0.007);
  float n = fbm(p);
  float n2 = fbm(p*3.1 + 4.0);
  float d = smoothstep(0.5 - uStorm*0.15, 0.85, n + (n2-0.5)*0.25);
  float a = d * 0.82 * uFade;
  if(a < 0.003) discard;
  float shade = 0.65 + 0.35*smoothstep(0.4, 0.9, n2);
  vec3 col = mix(uSky*0.9, uSunCol*0.55 + uSky*0.4, shade) * (1.0 - uStorm*0.55);
  frag = vec4(col, a);
}`
