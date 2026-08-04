import React, {FormEvent, useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import './styles.css';

type Device={id:number;name:string;hostname:string;os:string;architecture:string;agent_version:string;last_seen:string|null;status:string;warning_disk_percent:number;critical_disk_percent:number;failed_ports:number;metric:null|{cpu_percent:number;memory_percent:number;uptime_seconds:number;network_sent:number;network_recv:number};disks:{mountpoint:string;filesystem:string;total:number;used:number;free:number;percent:number}[]};
type PortCheck={id:number;name:string;host:string;port:number;protocol:'tcp'|'udp';timeout_seconds:number;udp_payload:string;expect_response:boolean;enabled:boolean;latest:null|{is_up:boolean;latency_ms:number;error:string}};
type Detail=Device&{history:{recorded_at:string;cpu_percent:number;memory_percent:number}[];port_checks:PortCheck[]};
const api=async(path:string,options:RequestInit={})=>{const r=await fetch(path,{credentials:'include',headers:{'Content-Type':'application/json',...(options.headers||{})},...options});if(!r.ok){let m='Request failed';try{m=(await r.json()).detail||m}catch{}throw new Error(m)}return r.status===204?null:r.json()};
const fmtBytes=(n:number)=>{const u=['B','KB','MB','GB','TB'];let i=0;while(n>=1024&&i<u.length-1){n/=1024;i++}return `${n.toFixed(i?1:0)} ${u[i]}`};
const uptime=(s:number)=>`${Math.floor(s/86400)}d ${Math.floor((s%86400)/3600)}h`;
const formatChartDate=(value:unknown, mode:'time'|'full'='full')=>{
  if(typeof value!=='string'&&typeof value!=='number'&&!(value instanceof Date))return '';
  const date=new Date(value);
  if(Number.isNaN(date.getTime()))return '';
  return mode==='time'?date.toLocaleTimeString([], {hour:'2-digit',minute:'2-digit'}):date.toLocaleString();
};
function Login({onLogin}:{onLogin:()=>void}){const [username,setUsername]=useState('admin'),[password,setPassword]=useState(''),[error,setError]=useState('');async function submit(e:FormEvent){e.preventDefault();setError('');try{await api('/api/auth/login',{method:'POST',body:JSON.stringify({username,password})});onLogin()}catch(e){setError((e as Error).message)}}return <main className="login-shell"><section className="login-card"><div className="brand-mark">E</div><p className="eyebrow">EVOLUTION IT SOLUTIONS</p><h1>Infrastructure clarity,<br/>without the noise.</h1><p className="muted">Sign in to your EITS monitoring console.</p><form onSubmit={submit}><label>Username<input value={username} onChange={e=>setUsername(e.target.value)} autoComplete="username"/></label><label>Password<input type="password" value={password} onChange={e=>setPassword(e.target.value)} autoComplete="current-password" autoFocus/></label>{error&&<p className="error">{error}</p>}<button>Sign in</button></form></section></main>}
function Meter({value}:{value:number}){return <div className="meter"><span style={{width:`${Math.min(100,value)}%`}}/></div>}
function Dashboard({logout}:{logout:()=>void}){const [data,setData]=useState<any>(null),[selected,setSelected]=useState<number|null>(null);const load=()=>api('/api/dashboard').then(setData).catch(()=>logout());useEffect(()=>{load();const id=setInterval(load,15000);return()=>clearInterval(id)},[]);if(!data)return <div className="loading">Loading EITS Monitor…</div>;if(selected)return <DeviceView id={selected} back={()=>{setSelected(null);load()}}/>;return <div className="app"><header><div><p className="eyebrow">EITS MONITOR</p><h1>System overview</h1></div><button className="ghost" onClick={logout}>Sign out</button></header><section className="kpis"><Kpi label="Total devices" value={data.total}/><Kpi label="Healthy" value={data.counts.healthy}/><Kpi label="Warning" value={data.counts.warning}/><Kpi label="Critical" value={data.counts.critical}/><Kpi label="Offline" value={data.counts.offline}/></section><section><div className="section-title"><div><p className="eyebrow">LIVE INVENTORY</p><h2>Monitored devices</h2></div><span className="muted">Refreshes every 15 seconds</span></div><div className="device-grid">{data.devices.map((d:Device)=><article className="device-card" key={d.id} onClick={()=>setSelected(d.id)}><div className="device-head"><div><h3>{d.name}</h3><p>{d.hostname||'Awaiting hostname'} · {d.os}/{d.architecture}</p></div><span className={`status ${d.status}`}>{d.status}</span></div><div className="metric-pair"><div><span>CPU</span><strong>{d.metric?.cpu_percent.toFixed(1)??'—'}%</strong><Meter value={d.metric?.cpu_percent??0}/></div><div><span>Memory</span><strong>{d.metric?.memory_percent.toFixed(1)??'—'}%</strong><Meter value={d.metric?.memory_percent??0}/></div></div><div className="card-foot"><span>{d.disks.length} disks</span><span>{d.failed_ports} failed ports</span><span>{d.last_seen ? formatChartDate(d.last_seen, 'time') : 'Never seen'}</span></div></article>)}</div></section></div>}
function Kpi({label,value}:{label:string;value:number}){return <article className="kpi"><span>{label}</span><strong>{value}</strong></article>}
function DeviceView({id,back}:{id:number;back:()=>void}){
  const [d,setD]=useState<Detail|null>(null);
  const [error,setError]=useState('');
  const [savingCheck,setSavingCheck]=useState(false);
  const [protocol,setProtocol]=useState<'tcp'|'udp'>('tcp');
  const [expectResponse,setExpectResponse]=useState(false);
  const load=async()=>{try{setD(await api(`/api/devices/${id}`));setError('')}catch(e){setError((e as Error).message)}};
  useEffect(()=>{load();const timer=setInterval(load,15000);return()=>clearInterval(timer)},[id]);
  async function addPort(e:FormEvent<HTMLFormElement>){
    e.preventDefault();
    const form=e.currentTarget;
    const f=new FormData(form);
    setSavingCheck(true);setError('');
    try{
      const created=await api(`/api/devices/${id}/port-checks`,{method:'POST',body:JSON.stringify({
        name:f.get('name'),host:f.get('host'),port:Number(f.get('port')),protocol,
        timeout_seconds:Number(f.get('timeout_seconds')||3),udp_payload:String(f.get('udp_payload')||''),expect_response:expectResponse
      })}) as PortCheck;
      setD(current=>current?{...current,port_checks:[...current.port_checks,created].sort((a,b)=>a.name.localeCompare(b.name))}:current);
      form.reset();setProtocol('tcp');setExpectResponse(false);
      await load();
    }catch(e){setError((e as Error).message)}finally{setSavingCheck(false)}
  }
  async function removeCheck(checkId:number){
    setD(current=>current?{...current,port_checks:current.port_checks.filter(c=>c.id!==checkId)}:current);
    try{await api(`/api/devices/${id}/port-checks/${checkId}`,{method:'DELETE'});await load()}catch(e){setError((e as Error).message);await load()}
  }
  async function saveThresholds(e:FormEvent<HTMLFormElement>){e.preventDefault();const f=new FormData(e.currentTarget);await api(`/api/devices/${id}/thresholds`,{method:'PATCH',body:JSON.stringify({warning_disk_percent:Number(f.get('warning')),critical_disk_percent:Number(f.get('critical'))})});await load()}
  if(error&&!d)return <div className="loading">{error}</div>;
  if(!d)return <div className="loading">Loading device…</div>;
  return <div className="app"><header><div><button className="back" onClick={back}>← Overview</button><p className="eyebrow">DEVICE DETAIL</p><h1>{d.name}</h1><p className="muted">{d.hostname} · {d.os}/{d.architecture} · agent {d.agent_version}</p></div><span className={`status ${d.status}`}>{d.status}</span></header>
  {error&&<p className="error notice">{error}</p>}
  <section className="kpis"><Kpi label="CPU" value={Math.round(d.metric?.cpu_percent||0)}/><Kpi label="Memory %" value={Math.round(d.metric?.memory_percent||0)}/><Kpi label="Uptime days" value={Math.floor((d.metric?.uptime_seconds||0)/86400)}/><Kpi label="Check failures" value={d.failed_ports}/></section>
  <section className="panel"><h2>Last hour trend</h2><div className="chart"><ResponsiveContainer width="100%" height="100%"><AreaChart data={d.history}><CartesianGrid strokeDasharray="3 3"/><XAxis dataKey="recorded_at" tickFormatter={(value)=>formatChartDate(value==null?'':String(value),'time')}/><YAxis domain={[0,100]}/><Tooltip labelFormatter={(value)=>formatChartDate(value==null?'':String(value))}/><Area type="monotone" dataKey="cpu_percent" fillOpacity={.2}/><Area type="monotone" dataKey="memory_percent" fillOpacity={.1}/></AreaChart></ResponsiveContainer></div></section>
  <div className="two-col"><section className="panel"><h2>Disks</h2>{d.disks.map(x=><div className="disk" key={x.mountpoint}><div><strong>{x.mountpoint}</strong><span>{fmtBytes(x.free)} free of {fmtBytes(x.total)}</span></div><b>{x.percent.toFixed(1)}%</b><Meter value={x.percent}/></div>)}<form className="inline-form" onSubmit={saveThresholds}><input name="warning" type="number" defaultValue={d.warning_disk_percent}/><input name="critical" type="number" defaultValue={d.critical_disk_percent}/><button>Save thresholds</button></form></section>
  <section className="panel"><div className="panel-heading"><div><h2>Network checks</h2><p className="muted">TCP connection checks and UDP probes</p></div></div>{d.port_checks.map(c=><div className="port" key={c.id}><div><strong>{c.name} <small className="protocol-badge">{c.protocol.toUpperCase()}</small></strong><span>{c.host}:{c.port}{c.protocol==='udp'&&c.expect_response?' · response required':''}</span></div><span className={`status ${c.latest?.is_up?'healthy':c.latest?'critical':'unknown'}`}>{c.latest?c.latest.is_up?'up':'down':'pending'}</span><button className="danger" onClick={()=>removeCheck(c.id)} aria-label={`Delete ${c.name}`}>×</button></div>)}
  <form className="port-form network-check-form" onSubmit={addPort}><input name="name" placeholder="Service name" required/><input name="host" placeholder="10.0.0.4" required/><input name="port" type="number" min="1" max="65535" placeholder="443" required/><select name="protocol" value={protocol} onChange={e=>setProtocol(e.target.value as 'tcp'|'udp')}><option value="tcp">TCP</option><option value="udp">UDP</option></select><input name="timeout_seconds" type="number" min="1" max="30" defaultValue="3" title="Timeout seconds"/>{protocol==='udp'&&<><input name="udp_payload" placeholder="Optional UDP payload"/><label className="check-option"><input type="checkbox" checked={expectResponse} onChange={e=>setExpectResponse(e.target.checked)}/>Require response</label></>}<button disabled={savingCheck}>{savingCheck?'Adding…':'Add check'}</button></form>
  {protocol==='udp'&&<p className="udp-note">UDP has no handshake. Without “Require response”, a successful send with no explicit rejection is treated as up. Use a service-specific payload and require a response for stronger validation.</p>}</section></div></div>
}
function App(){const [auth,setAuth]=useState<boolean|null>(null);useEffect(()=>{api('/api/auth/me').then(()=>setAuth(true)).catch(()=>setAuth(false))},[]);if(auth===null)return <div className="loading">Starting EITS Monitor…</div>;if(!auth)return <Login onLogin={()=>setAuth(true)}/>;return <Dashboard logout={async()=>{await api('/api/auth/logout',{method:'POST'});setAuth(false)}}/>}createRoot(document.getElementById('root')!).render(<React.StrictMode><App/></React.StrictMode>);
