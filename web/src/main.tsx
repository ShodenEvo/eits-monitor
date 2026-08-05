import React, {FormEvent, useEffect, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import './styles.css';

type Inventory={collected_at:string;manufacturer:string;model:string;serial_number:string;device_type:string;os_name:string;os_version:string;os_build:string;kernel_version:string;last_os_update:string;cpu_vendor:string;cpu_model:string;cpu_physical_cores:number;cpu_logical_processors:number;total_memory_bytes:number;bios_version:string;gpus:{vendor:string;model:string;memory_bytes:number;driver_version:string}[]};
type Device={id:number;name:string;hostname:string;os:string;architecture:string;agent_version:string;last_seen:string|null;status:string;warning_disk_percent:number;critical_disk_percent:number;failed_ports:number;inventory:Inventory|null;metric:null|{cpu_percent:number;memory_percent:number;memory_total:number;memory_used:number;uptime_seconds:number;network_sent:number;network_recv:number};disks:{mountpoint:string;filesystem:string;total:number;used:number;free:number;percent:number}[]};
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
function Dashboard({logout}:{logout:()=>void}){const [data,setData]=useState<any>(null),[selected,setSelected]=useState<number|null>(null);const load=()=>api('/api/dashboard').then(setData).catch(()=>logout());useEffect(()=>{load();const id=setInterval(load,15000);return()=>clearInterval(id)},[]);if(!data)return <div className="loading">Loading EITS Monitor…</div>;if(selected)return <DeviceView id={selected} back={()=>{setSelected(null);load()}}/>;return <div className="app"><header><div><p className="eyebrow">EITS MONITOR</p><h1>System overview</h1></div><button className="ghost" onClick={logout}>Sign out</button></header><section className="kpis"><Kpi label="Total devices" value={data.total}/><Kpi label="Healthy" value={data.counts.healthy}/><Kpi label="Warning" value={data.counts.warning}/><Kpi label="Critical" value={data.counts.critical}/><Kpi label="Offline" value={data.counts.offline}/></section><section><div className="section-title"><div><p className="eyebrow">LIVE INVENTORY</p><h2>Monitored devices</h2></div><span className="muted">Refreshes every 15 seconds</span></div><div className="device-grid">{data.devices.map((d:Device)=>{const i=d.inventory;return <article className="device-card" key={d.id} onClick={()=>setSelected(d.id)}><div className="device-head"><div><h3>{d.name}</h3><p>{d.hostname||'Awaiting hostname'} · {i?.os_name||d.os}{i?.os_version?` ${i.os_version}`:''}</p></div><span className={`status ${d.status}`}>{d.status}</span></div><div className="inventory-strip"><div><span>Processor</span><strong>{i?.cpu_model||'Inventory pending'}</strong><small>{i?`${i.cpu_physical_cores} cores / ${i.cpu_logical_processors} threads`:'—'}</small></div><div><span>OS update</span><strong>{i?.last_os_update||'Not detected'}</strong><small>{i?.os_build?`Build ${i.os_build}`:d.architecture}</small></div></div><div className="metric-pair"><div><span>CPU usage</span><strong>{d.metric?.cpu_percent.toFixed(1)??'—'}%</strong><Meter value={d.metric?.cpu_percent??0}/></div><div><span>RAM</span><strong>{d.metric?`${fmtBytes(d.metric.memory_used)} / ${fmtBytes(d.metric.memory_total)}`:'—'}</strong><Meter value={d.metric?.memory_percent??0}/></div></div><div className="card-foot"><span>{d.disks.length} disks</span><span>{d.failed_ports} failed checks</span><span>{d.last_seen?formatChartDate(d.last_seen,'time'):'Never seen'}</span></div></article>})}</div></section></div>}
function Kpi({label,value}:{label:string;value:number}){return <article className="kpi"><span>{label}</span><strong>{value}</strong></article>}
function Info({label,value}:{label:string;value:string}){return <div className="info-item"><span>{label}</span><strong>{value||'Not available'}</strong></div>}
function DeviceView({id,back}:{id:number;back:()=>void}){
  const [d,setD]=useState<Detail|null>(null);
  const [error,setError]=useState('');
  const [savingCheck,setSavingCheck]=useState(false);
  const [editingCheck,setEditingCheck]=useState<PortCheck|null>(null);
  const [checkName,setCheckName]=useState('');
  const [checkPort,setCheckPort]=useState('');
  const [protocol,setProtocol]=useState<'tcp'|'udp'>('tcp');
  const [timeoutSeconds,setTimeoutSeconds]=useState('3');
  const [udpPayload,setUdpPayload]=useState('');
  const [expectResponse,setExpectResponse]=useState(false);
  const load=async()=>{try{setD(await api(`/api/devices/${id}`));setError('')}catch(e){setError((e as Error).message)}};
  useEffect(()=>{load();const timer=setInterval(load,15000);return()=>clearInterval(timer)},[id]);
  function resetCheckForm(){
    setEditingCheck(null);setCheckName('');setCheckPort('');setProtocol('tcp');setTimeoutSeconds('3');setUdpPayload('');setExpectResponse(false);
  }
  function beginEdit(check:PortCheck){
    setEditingCheck(check);setCheckName(check.name);setCheckPort(String(check.port));setProtocol(check.protocol);setTimeoutSeconds(String(check.timeout_seconds));setUdpPayload(check.udp_payload||'');setExpectResponse(check.expect_response);setError('');
  }
  async function savePort(e:FormEvent<HTMLFormElement>){
    e.preventDefault();setSavingCheck(true);setError('');
    const payload={name:checkName.trim(),host:'127.0.0.1',port:Number(checkPort),protocol,timeout_seconds:Number(timeoutSeconds||3),udp_payload:protocol==='udp'?udpPayload:'',expect_response:protocol==='udp'&&expectResponse};
    try{
      if(editingCheck){
        await api(`/api/devices/${id}/port-checks/${editingCheck.id}`,{method:'PATCH',body:JSON.stringify(payload)});
      }else{
        await api(`/api/devices/${id}/port-checks`,{method:'POST',body:JSON.stringify(payload)});
      }
      resetCheckForm();await load();
    }catch(e){setError((e as Error).message)}finally{setSavingCheck(false)}
  }
  async function removeCheck(checkId:number){
    setD(current=>current?{...current,port_checks:current.port_checks.filter(c=>c.id!==checkId)}:current);
    try{await api(`/api/devices/${id}/port-checks/${checkId}`,{method:'DELETE'});if(editingCheck?.id===checkId)resetCheckForm();await load()}catch(e){setError((e as Error).message);await load()}
  }
  async function saveThresholds(e:FormEvent<HTMLFormElement>){e.preventDefault();const f=new FormData(e.currentTarget);await api(`/api/devices/${id}/thresholds`,{method:'PATCH',body:JSON.stringify({warning_disk_percent:Number(f.get('warning')),critical_disk_percent:Number(f.get('critical'))})});await load()}
  if(error&&!d)return <div className="loading">{error}</div>;
  if(!d)return <div className="loading">Loading device…</div>;
  return <div className="app"><header><div><button className="back" onClick={back}>← Overview</button><p className="eyebrow">DEVICE DETAIL</p><h1>{d.name}</h1><p className="muted">{d.hostname} · {d.os}/{d.architecture} · agent {d.agent_version}</p></div><span className={`status ${d.status}`}>{d.status}</span></header>
  {error&&<p className="error notice">{error}</p>}
  <section className="kpis"><Kpi label="CPU %" value={Math.round(d.metric?.cpu_percent||0)}/><Kpi label="Memory %" value={Math.round(d.metric?.memory_percent||0)}/><Kpi label="Uptime days" value={Math.floor((d.metric?.uptime_seconds||0)/86400)}/><Kpi label="Check failures" value={d.failed_ports}/></section>
  <section className="panel inventory-panel"><div className="panel-heading"><div><h2>Hardware & operating system</h2><p className="muted">Inventory collected {d.inventory?.collected_at?formatChartDate(d.inventory.collected_at):'not yet'}</p></div></div>{d.inventory?<div className="inventory-grid"><Info label="System" value={[d.inventory.manufacturer,d.inventory.model].filter(Boolean).join(' ')||'Unknown'}/><Info label="Device type" value={d.inventory.device_type}/><Info label="Serial number" value={d.inventory.serial_number||'Not available'}/><Info label="Operating system" value={[d.inventory.os_name,d.inventory.os_version].filter(Boolean).join(' ')}/><Info label="OS build" value={d.inventory.os_build||'Not available'}/><Info label="Latest detected update" value={d.inventory.last_os_update||'Not detected'}/><Info label="Processor" value={d.inventory.cpu_model||'Unknown'}/><Info label="CPU topology" value={`${d.inventory.cpu_physical_cores} cores / ${d.inventory.cpu_logical_processors} threads`}/><Info label="Installed RAM" value={fmtBytes(d.inventory.total_memory_bytes||d.metric?.memory_total||0)}/><Info label="BIOS" value={d.inventory.bios_version||'Not available'}/><Info label="Kernel" value={d.inventory.kernel_version||'Not available'}/><Info label="Graphics" value={d.inventory.gpus?.length?d.inventory.gpus.map(g=>g.model).join(', '):'No GPU detected'}/></div>:<p className="muted">Waiting for the agent to submit its first inventory scan.</p>}</section>
  <section className="panel"><h2>Last hour trend</h2><div className="chart"><ResponsiveContainer width="100%" height="100%"><AreaChart data={d.history}><CartesianGrid strokeDasharray="3 3"/><XAxis dataKey="recorded_at" tickFormatter={(value)=>formatChartDate(value==null?'':String(value),'time')}/><YAxis domain={[0,100]}/><Tooltip labelFormatter={(value)=>formatChartDate(value==null?'':String(value))}/><Area type="monotone" dataKey="cpu_percent" fillOpacity={.2}/><Area type="monotone" dataKey="memory_percent" fillOpacity={.1}/></AreaChart></ResponsiveContainer></div></section>
  <div className="two-col"><section className="panel"><h2>Disks</h2>{d.disks.map(x=><div className="disk" key={x.mountpoint}><div><strong>{x.mountpoint}</strong><span>{fmtBytes(x.free)} free of {fmtBytes(x.total)}</span></div><b>{x.percent.toFixed(1)}%</b><Meter value={x.percent}/></div>)}<form className="inline-form" onSubmit={saveThresholds}><input name="warning" type="number" defaultValue={d.warning_disk_percent}/><input name="critical" type="number" defaultValue={d.critical_disk_percent}/><button>Save thresholds</button></form></section>
  <section className="panel"><div className="panel-heading"><div><h2>Network checks</h2><p className="muted">Checks run locally on this monitored device using 127.0.0.1.</p></div></div>
  <div className="check-list-header"><span>Service</span><span>Port</span><span>Protocol</span><span>Status</span><span>Actions</span></div>
  {d.port_checks.map(c=><div className="port check-row" key={c.id}><div><strong>{c.name}</strong>{c.protocol==='udp'&&c.expect_response&&<small>Response required</small>}</div><span className="check-port">{c.port}</span><small className="protocol-badge">{c.protocol.toUpperCase()}</small><span className={`status ${c.latest?.is_up?'healthy':c.latest?'critical':'unknown'}`}>{c.latest?c.latest.is_up?'up':'down':'pending'}</span><div className="check-actions"><button className="secondary" type="button" onClick={()=>beginEdit(c)}>Edit</button><button className="danger" type="button" onClick={()=>removeCheck(c.id)} aria-label={`Delete ${c.name}`}>×</button></div></div>)}
  <form className="network-check-form" onSubmit={savePort}>
    <label><span>Service name</span><input value={checkName} onChange={e=>setCheckName(e.target.value)} placeholder="OCR service" required/></label>
    <label><span>Port</span><input value={checkPort} onChange={e=>setCheckPort(e.target.value)} type="number" min="1" max="65535" placeholder="8095" required/></label>
    <label><span>Protocol</span><select value={protocol} onChange={e=>setProtocol(e.target.value as 'tcp'|'udp')}><option value="tcp">TCP</option><option value="udp">UDP</option></select></label>
    <label><span>Timeout (seconds)</span><input value={timeoutSeconds} onChange={e=>setTimeoutSeconds(e.target.value)} type="number" min="1" max="30" required/></label>
    {protocol==='udp'&&<><label className="wide-field"><span>UDP payload (optional)</span><input value={udpPayload} onChange={e=>setUdpPayload(e.target.value)} placeholder="Optional payload"/></label><label className="check-option"><input type="checkbox" checked={expectResponse} onChange={e=>setExpectResponse(e.target.checked)}/><span>Require response</span></label></>}
    <div className="form-actions"><button disabled={savingCheck}>{savingCheck?(editingCheck?'Saving…':'Adding…'):(editingCheck?'Save changes':'Add check')}</button>{editingCheck&&<button className="secondary" type="button" onClick={resetCheckForm}>Cancel</button>}</div>
  </form>
  {protocol==='udp'&&<p className="udp-note">UDP has no handshake. Without “Require response”, a successful send with no explicit rejection is treated as up. Use a service-specific payload and require a response for stronger validation.</p>}</section></div></div>
}
function App(){const [auth,setAuth]=useState<boolean|null>(null);useEffect(()=>{api('/api/auth/me').then(()=>setAuth(true)).catch(()=>setAuth(false))},[]);if(auth===null)return <div className="loading">Starting EITS Monitor…</div>;if(!auth)return <Login onLogin={()=>setAuth(true)}/>;return <Dashboard logout={async()=>{await api('/api/auth/logout',{method:'POST'});setAuth(false)}}/>}createRoot(document.getElementById('root')!).render(<React.StrictMode><App/></React.StrictMode>);
