from pathlib import Path
import json
R=Path(__file__).resolve().parents[1]
M=[('lots','Lot','queued','running','completed'),('tools','Tool','idle','reserved','processing'),('chambers','Chamber','available','conditioned','occupied'),('recipes','Recipe','draft','qualified','released'),('reticles','Reticle','stored','checked_out','mounted'),('carriers','Carrier','empty','loaded','sealed'),('transport','Move','requested','assigned','delivered'),('dispatch','Decision','pending','committed','executed'),('reservations','Reservation','proposed','held','consumed'),('queues','QueueEntry','waiting','selected','started'),('qualifications','Qualification','scheduled','running','passed'),('maintenance','WorkWindow','planned','active','closed'),('contamination','ZoneState','clean','restricted','quarantined'),('holds','Hold','open','reviewing','released'),('rework','ReworkPlan','requested','approved','routed'),('sampling','SamplePlan','draft','armed','collected'),('routing','Route','draft','validated','active'),('batching','Batch','forming','sealed','launched'),('capacity','CapacitySlot','open','allocated','consumed'),('energy','EnergyWindow','forecast','limited','settled'),('purge','PurgeCycle','requested','running','verified'),('shifts','ShiftHandoff','prepared','accepted','closed'),('alarms','Alarm','raised','acknowledged','cleared'),('recovery','RecoveryPlan','detected','executing','restored'),('audit','AuditRecord','staged','committed','archived')]
def C(s):return ''.join(x.title() for x in s.split('_'))
def w(p,s):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(s)
w(R/'go.mod','module semiconductor-fab-lot-dispatch-service\n\ngo 1.23\n')
w(R/'internal/platform/errors.go','''package platform
import("errors";"fmt")
var(ErrNotFound=errors.New("resource not found");ErrConflict=errors.New("resource version conflict");ErrInvalidTransition=errors.New("invalid state transition");ErrCapacity=errors.New("capacity exhausted");ErrInvariant=errors.New("fab invariant violated"))
type OperationError struct{Operation,Resource,ID string;Err error}
func(e *OperationError)Error()string{return fmt.Sprintf("%s %s %s: %v",e.Operation,e.Resource,e.ID,e.Err)}
func(e *OperationError)Unwrap()error{return e.Err}
func Wrap(op,resource,id string,err error)error{if err==nil{return nil};return &OperationError{op,resource,id,err}}
''')
w(R/'internal/platform/clock.go','''package platform
import("sync";"time")
type Clock interface{Now()time.Time};type RealClock struct{};func(RealClock)Now()time.Time{return time.Now().UTC()}
type ManualClock struct{mu sync.RWMutex;now time.Time};func NewManualClock(t time.Time)*ManualClock{return &ManualClock{now:t.UTC()}};func(c *ManualClock)Now()time.Time{c.mu.RLock();defer c.mu.RUnlock();return c.now};func(c *ManualClock)Advance(d time.Duration){c.mu.Lock();c.now=c.now.Add(d);c.mu.Unlock()}
''')
w(R/'internal/platform/events.go','''package platform
import("context";"sync";"time")
type Event struct{Topic,Key string;Version int64;At time.Time;Attributes map[string]string};type EventSink interface{Publish(context.Context,Event)error}
type MemoryEventSink struct{mu sync.RWMutex;events []Event};func(s *MemoryEventSink)Publish(ctx context.Context,e Event)error{select{case<-ctx.Done():return ctx.Err();default:};s.mu.Lock();defer s.mu.Unlock();e.Attributes=CloneMap(e.Attributes);s.events=append(s.events,e);return nil};func(s *MemoryEventSink)Events()[]Event{s.mu.RLock();defer s.mu.RUnlock();out:=make([]Event,len(s.events));copy(out,s.events);return out};func CloneMap(in map[string]string)map[string]string{out:=make(map[string]string,len(in));for k,v:=range in{out[k]=v};return out}
''')
for pkg,typ,a,b,c in M:
 d=R/'internal'/pkg
 w(d/'types.go',f'''package {pkg}
import "time"
type {typ} struct{{ID string `json:"id"`;FabID string `json:"fab_id"`;State string `json:"state"`;Priority int `json:"priority"`;Quantity int `json:"quantity"`;Version int64 `json:"version"`;ReadyAt time.Time `json:"ready_at"`;Deadline time.Time `json:"deadline"`;UpdatedAt time.Time `json:"updated_at"`;Constraints map[string]string `json:"constraints"`;Tags []string `json:"tags"`}}
const(State{C(a)}="{a}";State{C(b)}="{b}";State{C(c)}="{c}")
type Command struct{{ID string `json:"id"`;ExpectedVersion int64 `json:"expected_version"`;TargetState string `json:"target_state"`;Priority int `json:"priority"`;Quantity int `json:"quantity"`;Deadline time.Time `json:"deadline"`;Constraints map[string]string `json:"constraints"`;Reason string `json:"reason"`}}
type Candidate struct{{ID string `json:"id"`;Score int64 `json:"score"`;Feasible bool `json:"feasible"`;Reasons []string `json:"reasons"`;Version int64 `json:"version"`}}
type Snapshot struct{{Items []{typ} `json:"items"`;Version int64 `json:"version"`;GeneratedAt time.Time `json:"generated_at"`}}
func(x {typ})Clone(){typ}{{x.Constraints=cloneMap(x.Constraints);x.Tags=append([]string(nil),x.Tags...);return x}}
func cloneMap(in map[string]string)map[string]string{{out:=make(map[string]string,len(in));for k,v:=range in{{out[k]=v}};return out}}
''')
 w(d/'repository.go',f'''package {pkg}
import("context";"sort";"sync";"semiconductor-fab-lot-dispatch-service/internal/platform")
type Repository struct{{mu sync.RWMutex;items map[string]{typ};version int64}}
func NewRepository()*Repository{{return &Repository{{items:make(map[string]{typ})}}}}
func(r *Repository)Create(ctx context.Context,item {typ})({typ},error){{if err:=ctx.Err();err!=nil{{return {typ}{{}},err}};r.mu.Lock();defer r.mu.Unlock();if _,ok:=r.items[item.ID];ok{{return {typ}{{}},platform.Wrap("create","{pkg}",item.ID,platform.ErrConflict)}};r.version++;item.Version=r.version;r.items[item.ID]=item.Clone();return item.Clone(),nil}}
func(r *Repository)Get(ctx context.Context,id string)({typ},error){{if err:=ctx.Err();err!=nil{{return {typ}{{}},err}};r.mu.RLock();defer r.mu.RUnlock();item,ok:=r.items[id];if !ok{{return {typ}{{}},platform.Wrap("get","{pkg}",id,platform.ErrNotFound)}};return item.Clone(),nil}}
func(r *Repository)Update(ctx context.Context,id string,expected int64,fn func(*{typ})error)({typ},error){{if err:=ctx.Err();err!=nil{{return {typ}{{}},err}};r.mu.Lock();defer r.mu.Unlock();cur,ok:=r.items[id];if !ok{{return {typ}{{}},platform.Wrap("update","{pkg}",id,platform.ErrNotFound)}};if expected>0&&cur.Version!=expected{{return {typ}{{}},platform.Wrap("update","{pkg}",id,platform.ErrConflict)}};next:=cur.Clone();if err:=fn(&next);err!=nil{{return {typ}{{}},err}};r.version++;next.Version=r.version;r.items[id]=next.Clone();return next.Clone(),nil}}
func(r *Repository)Restore(ctx context.Context,item {typ},expected int64)({typ},error){{return r.Update(ctx,item.ID,expected,func(next *{typ})error{{*next=item.Clone();return nil}})}}
func(r *Repository)List(ctx context.Context,fab string)([]{typ},error){{if err:=ctx.Err();err!=nil{{return nil,err}};r.mu.RLock();defer r.mu.RUnlock();out:=make([]{typ},0,len(r.items));for _,x:=range r.items{{if fab==""||x.FabID==fab{{out=append(out,x.Clone())}}}};sort.Slice(out,func(i,j int)bool{{if out[i].Priority==out[j].Priority{{return out[i].ID<out[j].ID}};return out[i].Priority>out[j].Priority}});return out,nil}}
func(r *Repository)Snapshot(ctx context.Context,fab string)(Snapshot,error){{items,err:=r.List(ctx,fab);if err!=nil{{return Snapshot{{}},err}};r.mu.RLock();v:=r.version;r.mu.RUnlock();return Snapshot{{Items:items,Version:v}},nil}}
func(r *Repository)Delete(ctx context.Context,id string,expected int64)error{{if err:=ctx.Err();err!=nil{{return err}};r.mu.Lock();defer r.mu.Unlock();x,ok:=r.items[id];if !ok{{return platform.Wrap("delete","{pkg}",id,platform.ErrNotFound)}};if expected>0&&x.Version!=expected{{return platform.Wrap("delete","{pkg}",id,platform.ErrConflict)}};delete(r.items,id);r.version++;return nil}}
''')
 w(d/'policy.go',f'''package {pkg}
import("fmt";"sort";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type Policy struct{{MaxQuantity int;MinimumPriority int;AllowedTransitions map[string]map[string]bool}}
func DefaultPolicy()Policy{{return Policy{{MaxQuantity:1000,AllowedTransitions:map[string]map[string]bool{{State{C(a)}:{{State{C(b)}:true}},State{C(b)}:{{State{C(c)}:true,State{C(a)}:true}},State{C(c)}:{{State{C(a)}:true}}}}}}}}
func(p Policy)Validate(x {typ})error{{if x.ID==""||x.FabID==""{{return fmt.Errorf("%w: id and fab required",platform.ErrInvariant)}};if x.Quantity<0||x.Quantity>p.MaxQuantity{{return fmt.Errorf("%w: quantity %d",platform.ErrCapacity,x.Quantity)}};if x.Priority<p.MinimumPriority{{return fmt.Errorf("%w: priority floor",platform.ErrInvariant)}};if !x.Deadline.IsZero()&&!x.ReadyAt.IsZero()&&x.Deadline.Before(x.ReadyAt){{return fmt.Errorf("%w: deadline before readiness",platform.ErrInvariant)}};return nil}}
func(p Policy)CanTransition(from,to string)bool{{return from==to||p.AllowedTransitions[from][to]}}
func(p Policy)Score(x {typ},now time.Time)Candidate{{reasons:=[]string{{}};score:=int64(x.Priority*1000+x.Quantity);ok:=true;if !x.ReadyAt.IsZero()&&x.ReadyAt.After(now){{ok=false;reasons=append(reasons,"not-ready")}};if !x.Deadline.IsZero(){{m:=int64(x.Deadline.Sub(now)/time.Minute);if m<0{{score+=500000;reasons=append(reasons,"late")}}else if m<120{{score+=100000-m;reasons=append(reasons,"due-soon")}}}};if x.State==State{C(c)}{{ok=false;reasons=append(reasons,"terminal")}};return Candidate{{ID:x.ID,Score:score,Feasible:ok,Reasons:reasons,Version:x.Version}}}}
func Rank(p Policy,items []{typ},now time.Time)[]Candidate{{out:=make([]Candidate,0,len(items));for _,x:=range items{{out=append(out,p.Score(x,now))}};sort.SliceStable(out,func(i,j int)bool{{if out[i].Feasible!=out[j].Feasible{{return out[i].Feasible}};if out[i].Score==out[j].Score{{return out[i].ID<out[j].ID}};return out[i].Score>out[j].Score}});return out}}
''')
 w(d/'service.go',f'''package {pkg}
import("context";"fmt";"semiconductor-fab-lot-dispatch-service/internal/platform")
type Service struct{{repo *Repository;policy Policy;clock platform.Clock;events platform.EventSink}}
func NewService(r *Repository,p Policy,c platform.Clock,e platform.EventSink)*Service{{return &Service{{repo:r,policy:p,clock:c,events:e}}}}
func(s *Service)Register(ctx context.Context,x {typ})({typ},error){{if x.State==""{{x.State=State{C(a)}}};now:=s.clock.Now();if x.ReadyAt.IsZero(){{x.ReadyAt=now}};x.UpdatedAt=now;if err:=s.policy.Validate(x);err!=nil{{return {typ}{{}},platform.Wrap("register","{pkg}",x.ID,err)}};saved,err:=s.repo.Create(ctx,x);if err!=nil{{return {typ}{{}},err}};if err=s.events.Publish(ctx,platform.Event{{Topic:"{pkg}.registered",Key:saved.ID,Version:saved.Version,At:now,Attributes:platform.CloneMap(saved.Constraints)}});err!=nil{{_ = s.repo.Delete(context.Background(),saved.ID,saved.Version);return {typ}{{}},platform.Wrap("publish","{pkg}",saved.ID,err)}};return saved,nil}}
func(s *Service)Apply(ctx context.Context,cmd Command)({typ},error){{before,err:=s.repo.Get(ctx,cmd.ID);if err!=nil{{return {typ}{{}},err}};target:=cmd.TargetState;if target==""{{target=before.State}};if !s.policy.CanTransition(before.State,target){{return {typ}{{}},platform.Wrap("transition","{pkg}",cmd.ID,platform.ErrInvalidTransition)}};updated,err:=s.repo.Update(ctx,cmd.ID,cmd.ExpectedVersion,func(next *{typ})error{{next.State=target;if cmd.Priority!=0{{next.Priority=cmd.Priority}};if cmd.Quantity!=0{{next.Quantity=cmd.Quantity}};if !cmd.Deadline.IsZero(){{next.Deadline=cmd.Deadline}};for k,v:=range cmd.Constraints{{if next.Constraints==nil{{next.Constraints=map[string]string{{}}}};next.Constraints[k]=v}};next.UpdatedAt=s.clock.Now();return s.policy.Validate(*next)}});if err!=nil{{return {typ}{{}},err}};attrs:=map[string]string{{"from":before.State,"to":updated.State,"reason":cmd.Reason}};if err=s.events.Publish(ctx,platform.Event{{Topic:"{pkg}.transitioned",Key:updated.ID,Version:updated.Version,At:updated.UpdatedAt,Attributes:attrs}});err!=nil{{_,rb:=s.repo.Restore(context.Background(),before,updated.Version);if rb!=nil{{return {typ}{{}},fmt.Errorf("publish: %w; rollback: %v",err,rb)}};return {typ}{{}},platform.Wrap("publish","{pkg}",updated.ID,err)}};return updated,nil}}
func(s *Service)Candidates(ctx context.Context,fab string)([]Candidate,error){{items,err:=s.repo.List(ctx,fab);if err!=nil{{return nil,err}};return Rank(s.policy,items,s.clock.Now()),nil}}
func(s *Service)ReserveBest(ctx context.Context,fab,target string)(Candidate,{typ},error){{ranked,err:=s.Candidates(ctx,fab);if err!=nil{{return Candidate{{}},{typ}{{}},err}};for _,c:=range ranked{{if !c.Feasible{{continue}};x,e:=s.Apply(ctx,Command{{ID:c.ID,ExpectedVersion:c.Version,TargetState:target,Reason:"dispatch-reservation"}});if e==nil{{return c,x,nil}};if ctx.Err()!=nil{{return Candidate{{}},{typ}{{}},ctx.Err()}}}};return Candidate{{}},{typ}{{}},platform.Wrap("reserve","{pkg}",fab,platform.ErrCapacity)}}
func(s *Service)Snapshot(ctx context.Context,fab string)(Snapshot,error){{x,err:=s.repo.Snapshot(ctx,fab);if err!=nil{{return Snapshot{{}},err}};x.GeneratedAt=s.clock.Now();return x,nil}}
''')
 w(d/'service_test.go',f'''package {pkg}
import("context";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
func TestServiceLifecycle(t *testing.T){{clock:=platform.NewManualClock(time.Date(2026,8,19,8,0,0,0,time.UTC));svc:=NewService(NewRepository(),DefaultPolicy(),clock,&platform.MemoryEventSink{{}});x,err:=svc.Register(context.Background(),{typ}{{ID:"x-1",FabID:"fab-a",State:State{C(a)},Priority:7,Quantity:25,Deadline:clock.Now().Add(time.Hour),Constraints:map[string]string{{"area":"bay-1"}}}});if err!=nil{{t.Fatal(err)}};got,err:=svc.Apply(context.Background(),Command{{ID:x.ID,ExpectedVersion:x.Version,TargetState:State{C(b)},Reason:"test"}});if err!=nil{{t.Fatal(err)}};if got.State!=State{C(b)}{{t.Fatalf("state=%s",got.State)}};ranked,err:=svc.Candidates(context.Background(),"fab-a");if err!=nil||len(ranked)!=1{{t.Fatalf("rank=%v err=%v",ranked,err)}}}}
func TestRepositoryRejectsStaleVersion(t *testing.T){{clock:=platform.NewManualClock(time.Now());svc:=NewService(NewRepository(),DefaultPolicy(),clock,&platform.MemoryEventSink{{}});x,err:=svc.Register(context.Background(),{typ}{{ID:"x-2",FabID:"fab-a",State:State{C(a)},Quantity:1}});if err!=nil{{t.Fatal(err)}};if _,err=svc.Apply(context.Background(),Command{{ID:x.ID,ExpectedVersion:x.Version+10,TargetState:State{C(b)}}});err==nil{{t.Fatal("expected conflict")}}}}
''')
# app files
w(R/'internal/api/server.go','''package api
import("encoding/json";"errors";"log/slog";"net/http";"strings";"time";"semiconductor-fab-lot-dispatch-service/internal/dispatch";"semiconductor-fab-lot-dispatch-service/internal/lots";"semiconductor-fab-lot-dispatch-service/internal/platform";"semiconductor-fab-lot-dispatch-service/internal/tools")
type Server struct{logger *slog.Logger;lots *lots.Service;tools *tools.Service;dispatch *dispatch.Service;started time.Time};func NewServer(l *slog.Logger,ls *lots.Service,ts *tools.Service,ds *dispatch.Service)*Server{return &Server{logger:l,lots:ls,tools:ts,dispatch:ds,started:time.Now().UTC()}}
func(s *Server)Handler()http.Handler{m:=http.NewServeMux();m.HandleFunc("GET /healthz",s.health);m.HandleFunc("GET /api/dashboard",s.dashboard);m.HandleFunc("POST /api/lots",s.createLot);m.HandleFunc("POST /api/lots/{id}/dispatch",s.dispatchLot);m.Handle("/",http.FileServer(http.Dir("frontend/dist")));return requestLog(s.logger,m)}
func(s *Server)health(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]any{"status":"ok","uptime_seconds":int(time.Since(s.started).Seconds())})}
func(s *Server)dashboard(w http.ResponseWriter,r *http.Request){a,e:=s.lots.Snapshot(r.Context(),r.URL.Query().Get("fab"));if e!=nil{writeError(w,e);return};b,e:=s.tools.Snapshot(r.Context(),r.URL.Query().Get("fab"));if e!=nil{writeError(w,e);return};writeJSON(w,200,map[string]any{"lots":a,"tools":b})}
func(s *Server)createLot(w http.ResponseWriter,r *http.Request){var x lots.Lot;if e:=json.NewDecoder(r.Body).Decode(&x);e!=nil{writeJSON(w,400,map[string]string{"error":e.Error()});return};x,e:=s.lots.Register(r.Context(),x);if e!=nil{writeError(w,e);return};writeJSON(w,201,x)}
func(s *Server)dispatchLot(w http.ResponseWriter,r *http.Request){id:=r.PathValue("id");if strings.TrimSpace(id)==""{writeJSON(w,400,map[string]string{"error":"missing lot id"});return};var c lots.Command;if e:=json.NewDecoder(r.Body).Decode(&c);e!=nil{writeJSON(w,400,map[string]string{"error":e.Error()});return};c.ID=id;if c.TargetState==""{c.TargetState=lots.StateRunning};x,e:=s.lots.Apply(r.Context(),c);if e!=nil{writeError(w,e);return};writeJSON(w,200,x)}
func writeError(w http.ResponseWriter,e error){n:=500;switch{case errors.Is(e,platform.ErrNotFound):n=404;case errors.Is(e,platform.ErrConflict):n=409;case errors.Is(e,platform.ErrInvalidTransition),errors.Is(e,platform.ErrInvariant):n=422;case errors.Is(e,platform.ErrCapacity):n=429};writeJSON(w,n,map[string]string{"error":e.Error()})};func writeJSON(w http.ResponseWriter,n int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(n);_ = json.NewEncoder(w).Encode(v)};func requestLog(l *slog.Logger,next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){at:=time.Now();next.ServeHTTP(w,r);l.Info("http request","method",r.Method,"path",r.URL.Path,"duration",time.Since(at))})}
''')
w(R/'internal/api/server_test.go','''package api
import("io";"log/slog";"net/http/httptest";"testing";"semiconductor-fab-lot-dispatch-service/internal/dispatch";"semiconductor-fab-lot-dispatch-service/internal/lots";"semiconductor-fab-lot-dispatch-service/internal/platform";"semiconductor-fab-lot-dispatch-service/internal/tools")
func TestHealth(t *testing.T){c:=platform.RealClock{};e:=&platform.MemoryEventSink{};s:=NewServer(slog.New(slog.NewTextHandler(io.Discard,nil)),lots.NewService(lots.NewRepository(),lots.DefaultPolicy(),c,e),tools.NewService(tools.NewRepository(),tools.DefaultPolicy(),c,e),dispatch.NewService(dispatch.NewRepository(),dispatch.DefaultPolicy(),c,e));r:=httptest.NewRequest("GET","/healthz",nil);w:=httptest.NewRecorder();s.Handler().ServeHTTP(w,r);if w.Code!=200{t.Fatalf("status=%d",w.Code)}}
''')
w(R/'cmd/fab-dispatch/main.go','''package main
import("context";"log/slog";"net/http";"os";"os/signal";"syscall";"time";"semiconductor-fab-lot-dispatch-service/internal/api";"semiconductor-fab-lot-dispatch-service/internal/dispatch";"semiconductor-fab-lot-dispatch-service/internal/lots";"semiconductor-fab-lot-dispatch-service/internal/platform";"semiconductor-fab-lot-dispatch-service/internal/tools")
func main(){l:=slog.New(slog.NewJSONHandler(os.Stdout,nil));c:=platform.RealClock{};e:=&platform.MemoryEventSink{};s:=api.NewServer(l,lots.NewService(lots.NewRepository(),lots.DefaultPolicy(),c,e),tools.NewService(tools.NewRepository(),tools.DefaultPolicy(),c,e),dispatch.NewService(dispatch.NewRepository(),dispatch.DefaultPolicy(),c,e));h:=&http.Server{Addr:env("HTTP_ADDR",":8080"),Handler:s.Handler(),ReadHeaderTimeout:5*time.Second};ctx,stop:=signal.NotifyContext(context.Background(),syscall.SIGINT,syscall.SIGTERM);defer stop();go func(){l.Info("listening","addr",h.Addr);if err:=h.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{l.Error("serve", "error",err);os.Exit(1)}}();<-ctx.Done();x,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel();_ = h.Shutdown(x)};func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
''')
# frontend
w(R/'frontend/package.json',json.dumps({'name':'fab-dispatch-console','private':True,'version':'1.0.0','scripts':{'build':'node build.mjs'}},indent=2))
w(R/'frontend/build.mjs',"import {cpSync,mkdirSync} from 'node:fs';mkdirSync('dist',{recursive:true});for(const f of ['index.html','app.js','styles.css'])cpSync('src/'+f,'dist/'+f);\n")
w(R/'frontend/src/index.html','''<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Fab Dispatch</title><link rel="stylesheet" href="/styles.css"></head><body><main><header><div><p>SEMICONDUCTOR OPERATIONS</p><h1>Fab Lot Dispatch Control</h1></div><span id="health">connecting</span></header><section class="stats"><article><b id="lots">0</b><small>Active lots</small></article><article><b id="tools">0</b><small>Process tools</small></article><article><b>nominal</b><small>Dispatch risk</small></article></section><section class="panel"><h2>Control-plane snapshot</h2><pre id="snapshot">Loading…</pre></section></main><script type="module" src="/app.js"></script></body></html>''')
w(R/'frontend/src/app.js',"const h=document.querySelector('#health');async function refresh(){try{const [x,d]=await Promise.all([fetch('/healthz').then(r=>r.json()),fetch('/api/dashboard?fab=fab-a').then(r=>r.json())]);h.textContent=x.status;h.className='ok';lots.textContent=d.lots.items.length;tools.textContent=d.tools.items.length;snapshot.textContent=JSON.stringify(d,null,2)}catch(e){h.textContent='offline';snapshot.textContent=e.message}}refresh();setInterval(refresh,5000);\n")
w(R/'frontend/src/styles.css',''':root{font-family:Inter,system-ui;color:#e8f0ff;background:#08111f}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 85% 10%,#123e55 0,transparent 36%),#08111f;min-height:100vh}main{max-width:1100px;margin:auto;padding:64px 28px}header{display:flex;justify-content:space-between;align-items:flex-end;border-bottom:1px solid #29445a;padding-bottom:28px}header p{color:#5eead4;letter-spacing:.2em;font-size:12px}h1{font-size:clamp(36px,6vw,72px);line-height:.96;margin:12px 0 0;max-width:760px}#health{border:1px solid #64748b;border-radius:99px;padding:8px 14px}.ok{color:#5eead4}.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:18px;margin:34px 0}.stats article,.panel{background:#0d1b2b;border:1px solid #1d364b;border-radius:14px;padding:24px}.stats b{display:block;font-size:34px}.stats small{color:#8aa2b8}pre{overflow:auto;color:#b8cada;line-height:1.5}@media(max-width:700px){.stats{grid-template-columns:1fr}header{align-items:flex-start;gap:24px;flex-direction:column}}''')
w(R/'runtime_smoke.json',json.dumps({'mode':'service','start':['go','run','./cmd/fab-dispatch'],'ready_url':'http://127.0.0.1:8080/healthz','timeout_seconds':30},indent=2))
w(R/'README.md','''# Semiconductor Fab Lot Dispatch Service

A Go control-plane for wafer-lot dispatch in a semiconductor fabrication facility. The service models lot readiness, tools, chambers, recipes, reticles, carriers, AMHS transport, reservations, qualifications, maintenance, contamination zones, holds, rework, sampling, routing, batching, capacity, energy windows, purge cycles, shift handoff, alarms, recovery, and audit records.

## Architecture
- `cmd/fab-dispatch`: HTTP service entry point.
- `internal/api`: health, dashboard, lot registration and dispatch endpoints.
- `internal/<context>`: domain model, versioned repository, policy and transactional service.
- `internal/platform`: clocks, typed errors and event abstractions.
- `frontend`: dependency-free operations console.

## Run
```bash
npm --prefix frontend run build
go run ./cmd/fab-dispatch
```
The service listens on `:8080`; set `HTTP_ADDR` to override it.

## Test
```bash
go test ./...
go build ./...
```

## HTTP endpoints
- `GET /healthz`
- `GET /api/dashboard?fab=fab-a`
- `POST /api/lots`
- `POST /api/lots/{id}/dispatch`
''')
w(R/'.gitignore','frontend/node_modules/\n.DS_Store\n')
print(R)
