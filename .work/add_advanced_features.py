from pathlib import Path
R=Path(__file__).resolve().parents[1]
def w(rel,s):p=R/rel;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(s)
# 1 lots
w('internal/lots/lease_store.go',r'''package lots

import (
 "context"
 "sync"
 "time"
 "semiconductor-fab-lot-dispatch-service/internal/platform"
)
type Lease struct{LotID,Owner string;Generation int64;ExpiresAt time.Time}
type LeaseStore struct{mu sync.Mutex;leases map[string]Lease;generation int64}
func NewLeaseStore()*LeaseStore{return &LeaseStore{leases:map[string]Lease{}}}
func(s *LeaseStore)TryAcquire(ctx context.Context,lotID,owner string,now time.Time,ttl time.Duration)(Lease,error){
 if err:=ctx.Err();err!=nil{return Lease{},err};s.mu.Lock();defer s.mu.Unlock()
 if current,ok:=s.leases[lotID];ok&&current.ExpiresAt.After(now){return Lease{},platform.ErrConflict}
 s.generation++;lease:=Lease{LotID:lotID,Owner:owner,Generation:s.generation,ExpiresAt:now.Add(ttl)};s.leases[lotID]=lease;return lease,nil
}
func(s *LeaseStore)Release(ctx context.Context,lease Lease)error{if err:=ctx.Err();err!=nil{return err};s.mu.Lock();defer s.mu.Unlock();current,ok:=s.leases[lease.LotID];if !ok{return nil};if current.Generation!=lease.Generation||current.Owner!=lease.Owner{return platform.ErrConflict};delete(s.leases,lease.LotID);return nil}
func(s *LeaseStore)Active(lotID string,now time.Time)(Lease,bool){s.mu.Lock();defer s.mu.Unlock();x,ok:=s.leases[lotID];if ok&&!x.ExpiresAt.After(now){delete(s.leases,lotID);return Lease{},false};return x,ok}
''')
w('internal/lots/lease_coordinator.go',r'''package lots

import(
 "context"
 "time"
 "semiconductor-fab-lot-dispatch-service/internal/platform"
)
type LeaseCoordinator struct{store *LeaseStore;clock platform.Clock;events platform.EventSink;ttl time.Duration}
func NewLeaseCoordinator(store *LeaseStore,clock platform.Clock,events platform.EventSink,ttl time.Duration)*LeaseCoordinator{return &LeaseCoordinator{store:store,clock:clock,events:events,ttl:ttl}}
func(c *LeaseCoordinator)AcquireBest(ctx context.Context,owner string,candidates []Candidate)(Lease,error){
 if err:=ctx.Err();err!=nil{return Lease{},err}
 for _,candidate:=range candidates{if !candidate.Feasible{continue};lease,err:=c.store.TryAcquire(ctx,candidate.ID,owner,c.clock.Now(),c.ttl);if err==platform.ErrConflict{continue};if err!=nil{return Lease{},err}
  event:=platform.Event{Topic:"lots.lease-acquired",Key:lease.LotID,Version:lease.Generation,At:c.clock.Now(),Attributes:map[string]string{"owner":owner}}
  if err=c.events.Publish(ctx,event);err!=nil{rollbackCtx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();_ = c.store.Release(rollbackCtx,lease);return Lease{},platform.Wrap("publish","lot-lease",lease.LotID,err)}
  return lease,nil
 }
 return Lease{},platform.ErrCapacity
}
func(c *LeaseCoordinator)Release(ctx context.Context,lease Lease)error{if err:=c.store.Release(ctx,lease);err!=nil{return err};return c.events.Publish(ctx,platform.Event{Topic:"lots.lease-released",Key:lease.LotID,Version:lease.Generation,At:c.clock.Now(),Attributes:map[string]string{"owner":lease.Owner}})}
''')
w('internal/lots/lease_coordinator_test.go',r'''package lots
import("context";"errors";"sync";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type rejectingSink struct{};func(rejectingSink)Publish(context.Context,platform.Event)error{return errors.New("broker unavailable")}
func TestLeaseCoordinatorConcurrentSingleWinner(t *testing.T){clock:=platform.NewManualClock(time.Now());store:=NewLeaseStore();c:=NewLeaseCoordinator(store,clock,&platform.MemoryEventSink{},time.Minute);cand:=[]Candidate{{ID:"lot-9",Feasible:true,Score:99}};start:=make(chan struct{});var wg sync.WaitGroup;wins:=0;var mu sync.Mutex;for i:=0;i<12;i++{wg.Add(1);go func(){defer wg.Done();<-start;if _,err:=c.AcquireBest(context.Background(),"dispatcher",cand);err==nil{mu.Lock();wins++;mu.Unlock()}}()};close(start);wg.Wait();if wins!=1{t.Fatalf("wins=%d",wins)}}
func TestLeaseCoordinatorPublishFailureRollsBack(t *testing.T){clock:=platform.NewManualClock(time.Now());store:=NewLeaseStore();c:=NewLeaseCoordinator(store,clock,rejectingSink{},time.Minute);_,err:=c.AcquireBest(context.Background(),"dispatcher",[]Candidate{{ID:"lot-7",Feasible:true}});if err==nil{t.Fatal("expected error")};if _,ok:=store.Active("lot-7",clock.Now());ok{t.Fatal("lease leaked after publish failure")}}
''')
# 2 tools
w('internal/tools/reservation_table.go',r'''package tools
import("context";"sort";"sync";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ToolReservation struct{Owner string;ToolIDs []string;Generation int64}
type ReservationTable struct{mu sync.Mutex;owners map[string]string;generation int64}
func NewReservationTable()*ReservationTable{return &ReservationTable{owners:map[string]string{}}}
func(t *ReservationTable)ReserveAll(ctx context.Context,owner string,ids []string)(ToolReservation,error){if err:=ctx.Err();err!=nil{return ToolReservation{},err};keys:=append([]string(nil),ids...);sort.Strings(keys);keys=dedupe(keys);t.mu.Lock();defer t.mu.Unlock();for _,id:=range keys{if current:=t.owners[id];current!=""&&current!=owner{return ToolReservation{},platform.ErrConflict}};t.generation++;for _,id:=range keys{t.owners[id]=owner};return ToolReservation{Owner:owner,ToolIDs:keys,Generation:t.generation},nil}
func(t *ReservationTable)Release(ctx context.Context,r ToolReservation)error{if err:=ctx.Err();err!=nil{return err};t.mu.Lock();defer t.mu.Unlock();for _,id:=range r.ToolIDs{if current:=t.owners[id];current!=r.Owner{return platform.ErrConflict}};for _,id:=range r.ToolIDs{delete(t.owners,id)};return nil}
func(t *ReservationTable)Owner(id string)string{t.mu.Lock();defer t.mu.Unlock();return t.owners[id]}
func dedupe(in []string)[]string{if len(in)==0{return nil};out:=in[:1];for _,x:=range in[1:]{if x!=out[len(out)-1]{out=append(out,x)}};return out}
''')
w('internal/tools/reservation_service.go',r'''package tools
import("context";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ReservationService struct{table *ReservationTable;events platform.EventSink;clock platform.Clock}
func NewReservationService(t *ReservationTable,e platform.EventSink,c platform.Clock)*ReservationService{return &ReservationService{table:t,events:e,clock:c}}
func(s *ReservationService)ReserveRoute(ctx context.Context,owner string,primary,backup []string)(ToolReservation,error){ids:=append(append([]string(nil),primary...),backup...);r,err:=s.table.ReserveAll(ctx,owner,ids);if err!=nil{return ToolReservation{},platform.Wrap("reserve","tool-route",owner,err)};attrs:=map[string]string{"owner":owner,"tool_count":string(rune(len(r.ToolIDs)+'0'))};if err=s.events.Publish(ctx,platform.Event{Topic:"tools.route-reserved",Key:owner,Version:r.Generation,At:s.clock.Now(),Attributes:attrs});err!=nil{rbCtx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();_ = s.table.Release(rbCtx,r);return ToolReservation{},platform.Wrap("publish","tool-route",owner,err)};return r,nil}
func(s *ReservationService)ReleaseRoute(ctx context.Context,r ToolReservation)error{if err:=s.table.Release(ctx,r);err!=nil{return err};return s.events.Publish(ctx,platform.Event{Topic:"tools.route-released",Key:r.Owner,Version:r.Generation,At:s.clock.Now()})}
''')
w('internal/tools/reservation_service_test.go',r'''package tools
import("context";"sync";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
func TestReservationServiceOverlappingRoutesAreAtomic(t *testing.T){table:=NewReservationTable();svc:=NewReservationService(table,&platform.MemoryEventSink{},platform.NewManualClock(time.Now()));start:=make(chan struct{});var wg sync.WaitGroup;wins:=0;var mu sync.Mutex;for _,owner:=range []string{"alpha","beta"}{wg.Add(1);go func(owner string){defer wg.Done();<-start;if _,err:=svc.ReserveRoute(context.Background(),owner,[]string{"etch-1","clean-1"},[]string{"metrology-1"});err==nil{mu.Lock();wins++;mu.Unlock()}}(owner)};close(start);wg.Wait();if wins!=1{t.Fatalf("wins=%d",wins)};owner:=table.Owner("etch-1");if owner==""||table.Owner("clean-1")!=owner||table.Owner("metrology-1")!=owner{t.Fatal("partial route reservation")}}
''')
# 3 chambers
w('internal/chambers/telemetry.go',r'''package chambers
import("sync";"time")
type TelemetrySample struct{ChamberID string;At time.Time;Values map[string]float64;Labels []string}
type TelemetryWindow struct{mu sync.RWMutex;capacity int;samples []TelemetrySample}
func NewTelemetryWindow(capacity int)*TelemetryWindow{return &TelemetryWindow{capacity:capacity}}
func(w *TelemetryWindow)Add(sample TelemetrySample){w.mu.Lock();defer w.mu.Unlock();sample=cloneSample(sample);w.samples=append(w.samples,sample);if len(w.samples)>w.capacity{drop:=len(w.samples)-w.capacity;next:=make([]TelemetrySample,w.capacity);copy(next,w.samples[drop:]);w.samples=next}}
func(w *TelemetryWindow)Snapshot()[]TelemetrySample{w.mu.RLock();defer w.mu.RUnlock();out:=make([]TelemetrySample,len(w.samples));for i,x:=range w.samples{out[i]=cloneSample(x)};return out}
func cloneSample(x TelemetrySample)TelemetrySample{x.Values=cloneValues(x.Values);x.Labels=append([]string(nil),x.Labels...);return x}
func cloneValues(in map[string]float64)map[string]float64{out:=make(map[string]float64,len(in));for k,v:=range in{out[k]=v};return out}
''')
w('internal/chambers/telemetry_aggregate.go',r'''package chambers
import("sort";"time")
type ChamberSummary struct{ChamberID string;Latest time.Time;Maximums map[string]float64;Labels []string;Samples int}
func AggregateTelemetry(samples []TelemetrySample)[]ChamberSummary{byID:=map[string]*ChamberSummary{};for _,sample:=range samples{summary:=byID[sample.ChamberID];if summary==nil{summary=&ChamberSummary{ChamberID:sample.ChamberID,Maximums:map[string]float64{}};byID[sample.ChamberID]=summary};summary.Samples++;if sample.At.After(summary.Latest){summary.Latest=sample.At;summary.Labels=append([]string(nil),sample.Labels...)};for k,v:=range sample.Values{current,ok:=summary.Maximums[k];if !ok||v>current{summary.Maximums[k]=v}}};out:=make([]ChamberSummary,0,len(byID));for _,x:=range byID{copySummary:=*x;copySummary.Maximums=cloneValues(x.Maximums);copySummary.Labels=append([]string(nil),x.Labels...);out=append(out,copySummary)};sort.Slice(out,func(i,j int)bool{return out[i].ChamberID<out[j].ChamberID});return out}
''')
w('internal/chambers/telemetry_test.go',r'''package chambers
import("testing";"time")
func TestTelemetrySnapshotsDoNotShareMutableState(t *testing.T){w:=NewTelemetryWindow(3);input:=TelemetrySample{ChamberID:"c1",At:time.Now(),Values:map[string]float64{"pressure":1.2},Labels:[]string{"stable"}};w.Add(input);input.Values["pressure"]=99;input.Labels[0]="bad";a:=w.Snapshot();a[0].Values["pressure"]=77;a[0].Labels[0]="changed";b:=w.Snapshot();if b[0].Values["pressure"]!=1.2||b[0].Labels[0]!="stable"{t.Fatalf("snapshot polluted: %#v",b[0])};summary:=AggregateTelemetry(b);summary[0].Maximums["pressure"]=55;if AggregateTelemetry(w.Snapshot())[0].Maximums["pressure"]!=1.2{t.Fatal("aggregate shares storage")}}
''')
# 4 recipes
w('internal/recipes/release_errors.go',r'''package recipes
import("fmt";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ReleaseValidationError struct{RecipeID string;Missing []string;Err error}
func(e *ReleaseValidationError)Error()string{return fmt.Sprintf("recipe %s release rejected: missing %v: %v",e.RecipeID,e.Missing,e.Err)}
func(e *ReleaseValidationError)Unwrap()error{return e.Err}
func validateRelease(x Recipe)error{required:=[]string{"process_family","tool_group","revision","checksum"};missing:=make([]string,0);for _,key:=range required{if x.Constraints[key]==""{missing=append(missing,key)}};if len(missing)>0{return &ReleaseValidationError{RecipeID:x.ID,Missing:missing,Err:platform.ErrInvariant}};return nil}
''')
w('internal/recipes/release_workflow.go',r'''package recipes
import("context";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ReleaseWorkflow struct{repo *Repository;events platform.EventSink;clock platform.Clock}
func NewReleaseWorkflow(r *Repository,e platform.EventSink,c platform.Clock)*ReleaseWorkflow{return &ReleaseWorkflow{repo:r,events:e,clock:c}}
func(w *ReleaseWorkflow)Release(ctx context.Context,id string,expected int64)(Recipe,error){before,err:=w.repo.Get(ctx,id);if err!=nil{return Recipe{},err};if err=validateRelease(before);err!=nil{return Recipe{},platform.Wrap("validate-release","recipe",id,err)};updated,err:=w.repo.Update(ctx,id,expected,func(next *Recipe)error{if next.State!=StateQualified{return platform.ErrInvalidTransition};next.State=StateReleased;next.UpdatedAt=w.clock.Now();return nil});if err!=nil{return Recipe{},err};event:=platform.Event{Topic:"recipes.released",Key:id,Version:updated.Version,At:w.clock.Now(),Attributes:map[string]string{"revision":updated.Constraints["revision"]}};if err=w.events.Publish(ctx,event);err!=nil{rbCtx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();if _,rb:=w.repo.Restore(rbCtx,before,updated.Version);rb!=nil{return Recipe{},platform.Wrap("rollback-release","recipe",id,rb)};return Recipe{},platform.Wrap("publish-release","recipe",id,err)};return updated,nil}
''')
w('internal/recipes/release_workflow_test.go',r'''package recipes
import("context";"errors";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type recipeFailSink struct{};func(recipeFailSink)Publish(context.Context,platform.Event)error{return errors.New("registry down")}
func TestReleaseWorkflowPreservesTypedValidationAndRollsBack(t *testing.T){repo:=NewRepository();clock:=platform.NewManualClock(time.Now());bad,_:=repo.Create(context.Background(),Recipe{ID:"r-bad",FabID:"fab",State:StateQualified,Constraints:map[string]string{"revision":"1"}});wf:=NewReleaseWorkflow(repo,&platform.MemoryEventSink{},clock);_,err:=wf.Release(context.Background(),bad.ID,bad.Version);if !errors.Is(err,platform.ErrInvariant){t.Fatalf("typed error lost: %v",err)};good,_:=repo.Create(context.Background(),Recipe{ID:"r-good",FabID:"fab",State:StateQualified,Constraints:map[string]string{"process_family":"etch","tool_group":"g1","revision":"2","checksum":"abc"}});wf=NewReleaseWorkflow(repo,recipeFailSink{},clock);_,err=wf.Release(context.Background(),good.ID,good.Version);if err==nil{t.Fatal("expected publish error")};after,_:=repo.Get(context.Background(),good.ID);if after.State!=StateQualified{t.Fatalf("state leaked: %s",after.State)}}
''')
# 5 reticles
w('internal/reticles/mount_registry.go',r'''package reticles
import("context";"sync";"semiconductor-fab-lot-dispatch-service/internal/platform")
type Mount struct{ReticleID,SlotID,ToolID,Zone string;Generation int64}
type MountRegistry struct{mu sync.Mutex;byReticle map[string]Mount;bySlot map[string]Mount;generation int64}
func NewMountRegistry()*MountRegistry{return &MountRegistry{byReticle:map[string]Mount{},bySlot:map[string]Mount{}}}
func(r *MountRegistry)Mount(ctx context.Context,m Mount)(Mount,error){if err:=ctx.Err();err!=nil{return Mount{},err};r.mu.Lock();defer r.mu.Unlock();if r.byReticle==nil{r.byReticle=map[string]Mount{}};if r.bySlot==nil{r.bySlot=map[string]Mount{}};if _,ok:=r.byReticle[m.ReticleID];ok{return Mount{},platform.ErrConflict};if _,ok:=r.bySlot[m.SlotID];ok{return Mount{},platform.ErrConflict};r.generation++;m.Generation=r.generation;r.byReticle[m.ReticleID]=m;r.bySlot[m.SlotID]=m;return m,nil}
func(r *MountRegistry)Unmount(ctx context.Context,m Mount)error{if err:=ctx.Err();err!=nil{return err};r.mu.Lock();defer r.mu.Unlock();current,ok:=r.byReticle[m.ReticleID];if !ok{return nil};if current.Generation!=m.Generation{return platform.ErrConflict};delete(r.byReticle,m.ReticleID);delete(r.bySlot,m.SlotID);return nil}
func(r *MountRegistry)LookupSlot(id string)(Mount,bool){r.mu.Lock();defer r.mu.Unlock();x,ok:=r.bySlot[id];return x,ok}
''')
w('internal/reticles/mount_service.go',r'''package reticles
import("context";"strings";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type MountService struct{registry *MountRegistry;events platform.EventSink;clock platform.Clock}
func NewMountService(r *MountRegistry,e platform.EventSink,c platform.Clock)*MountService{return &MountService{registry:r,events:e,clock:c}}
func(s *MountService)MountForExposure(ctx context.Context,reticleID,toolID,slotID,zone string)(Mount,error){if strings.TrimSpace(reticleID)==""||strings.TrimSpace(toolID)==""||strings.TrimSpace(slotID)==""||strings.TrimSpace(zone)==""{return Mount{},platform.ErrInvariant};m,err:=s.registry.Mount(ctx,Mount{ReticleID:reticleID,ToolID:toolID,SlotID:slotID,Zone:zone});if err!=nil{return Mount{},platform.Wrap("mount","reticle",reticleID,err)};if err=s.events.Publish(ctx,platform.Event{Topic:"reticles.mounted",Key:reticleID,Version:m.Generation,At:s.clock.Now(),Attributes:map[string]string{"slot":slotID,"tool":toolID,"zone":zone}});err!=nil{rbCtx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();_ = s.registry.Unmount(rbCtx,m);return Mount{},platform.Wrap("publish","reticle",reticleID,err)};return m,nil}
''')
w('internal/reticles/mount_service_test.go',r'''package reticles
import("context";"sync";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
func TestMountServiceInitializesAndSerializesSlotOwnership(t *testing.T){registry:=&MountRegistry{};svc:=NewMountService(registry,&platform.MemoryEventSink{},platform.NewManualClock(time.Now()));start:=make(chan struct{});var wg sync.WaitGroup;wins:=0;var mu sync.Mutex;for _,id:=range []string{"rt-a","rt-b"}{wg.Add(1);go func(id string){defer wg.Done();<-start;if _,err:=svc.MountForExposure(context.Background(),id,"litho-1","slot-3","clean");err==nil{mu.Lock();wins++;mu.Unlock()}}(id)};close(start);wg.Wait();if wins!=1{t.Fatalf("wins=%d",wins)};if _,ok:=registry.LookupSlot("slot-3");!ok{t.Fatal("slot not recorded")}}
''')
# 6 transport
w('internal/transport/move_plan.go',r'''package transport
import("context";"sync")
type MoveStep struct{ID,From,To string};type StepResult struct{ID string;Err error}
type ResultLedger struct{mu sync.Mutex;results []StepResult}
func(l *ResultLedger)Append(r StepResult){l.mu.Lock();l.results=append(l.results,r);l.mu.Unlock()}
func(l *ResultLedger)Snapshot()[]StepResult{l.mu.Lock();defer l.mu.Unlock();out:=make([]StepResult,len(l.results));copy(out,l.results);return out}
type StepWorker func(context.Context,MoveStep)error
''')
w('internal/transport/executor.go',r'''package transport
import("context";"sync")
type Executor struct{Parallelism int}
func(e Executor)Execute(ctx context.Context,steps []MoveStep,worker StepWorker)([]StepResult,error){limit:=e.Parallelism;if limit<1{limit=1};runCtx,cancel:=context.WithCancel(ctx);defer cancel();jobs:=make(chan MoveStep);ledger:=&ResultLedger{};var wg sync.WaitGroup;for i:=0;i<limit;i++{wg.Add(1);go func(){defer wg.Done();for{select{case<-runCtx.Done():return;case step,ok:=<-jobs:if !ok{return};err:=worker(runCtx,step);ledger.Append(StepResult{ID:step.ID,Err:err});if err!=nil{cancel();return}}}}()};sendDone:=make(chan struct{});go func(){defer close(sendDone);defer close(jobs);for _,step:=range steps{select{case<-runCtx.Done():return;case jobs<-step:}}}();<-sendDone;wg.Wait();results:=ledger.Snapshot();if err:=ctx.Err();err!=nil{return results,err};for _,r:=range results{if r.Err!=nil{return results,r.Err}};return results,nil}
''')
w('internal/transport/executor_test.go',r'''package transport
import("context";"errors";"sync/atomic";"testing";"time")
func TestExecutorWaitsForWorkersOnCancellation(t *testing.T){ctx,cancel:=context.WithCancel(context.Background());var active atomic.Int32;started:=make(chan struct{},4);worker:=func(ctx context.Context,step MoveStep)error{active.Add(1);defer active.Add(-1);started<-struct{}{};<-ctx.Done();return ctx.Err()};done:=make(chan error,1);go func(){_,err:=Executor{Parallelism:4}.Execute(ctx,[]MoveStep{{ID:"1"},{ID:"2"},{ID:"3"},{ID:"4"}},worker);done<-err}();for i:=0;i<4;i++{<-started};cancel();select{case err:=<-done:if !errors.Is(err,context.Canceled){t.Fatalf("err=%v",err)};case<-time.After(time.Second):t.Fatal("executor did not return")};if active.Load()!=0{t.Fatalf("workers still active: %d",active.Load())}}
''')
# 7 dispatch
w('internal/dispatch/plan_cache.go',r'''package dispatch
import"sync"
type DispatchPlan struct{FabID string;Version int64;Assignments map[string]string;Order []string;Scores map[string]int64}
type PlanCache struct{mu sync.RWMutex;plans map[string]DispatchPlan}
func NewPlanCache()*PlanCache{return &PlanCache{plans:map[string]DispatchPlan{}}}
func(c *PlanCache)Publish(plan DispatchPlan){copyPlan:=clonePlan(plan);c.mu.Lock();c.plans[plan.FabID]=copyPlan;c.mu.Unlock()}
func(c *PlanCache)Snapshot(fab string)(DispatchPlan,bool){c.mu.RLock();plan,ok:=c.plans[fab];c.mu.RUnlock();if !ok{return DispatchPlan{},false};return clonePlan(plan),true}
func clonePlan(x DispatchPlan)DispatchPlan{x.Assignments=cloneAssignments(x.Assignments);x.Scores=cloneScores(x.Scores);x.Order=append([]string(nil),x.Order...);return x}
func cloneAssignments(in map[string]string)map[string]string{out:=make(map[string]string,len(in));for k,v:=range in{out[k]=v};return out}
func cloneScores(in map[string]int64)map[string]int64{out:=make(map[string]int64,len(in));for k,v:=range in{out[k]=v};return out}
''')
w('internal/dispatch/plan_builder.go',r'''package dispatch
import"sort"
type PlanInput struct{FabID string;Version int64;Lots []string;EligibleTools map[string][]string;Scores map[string]int64}
func BuildPlan(input PlanInput)DispatchPlan{lots:=append([]string(nil),input.Lots...);sort.SliceStable(lots,func(i,j int)bool{a,b:=input.Scores[lots[i]],input.Scores[lots[j]];if a==b{return lots[i]<lots[j]};return a>b});used:=map[string]bool{};assignments:=map[string]string{};order:=make([]string,0,len(lots));for _,lot:=range lots{tools:=append([]string(nil),input.EligibleTools[lot]...);sort.Strings(tools);for _,tool:=range tools{if !used[tool]{used[tool]=true;assignments[lot]=tool;order=append(order,lot);break}}};return DispatchPlan{FabID:input.FabID,Version:input.Version,Assignments:assignments,Order:order,Scores:cloneScores(input.Scores)}}
''')
w('internal/dispatch/plan_cache_test.go',r'''package dispatch
import("sync";"testing")
func TestPlanCachePublishesImmutableSnapshots(t *testing.T){cache:=NewPlanCache();input:=PlanInput{FabID:"fab",Version:1,Lots:[]string{"b","a"},EligibleTools:map[string][]string{"a":{"t1"},"b":{"t2"}},Scores:map[string]int64{"a":9,"b":5}};plan:=BuildPlan(input);cache.Publish(plan);plan.Assignments["a"]="corrupt";plan.Order[0]="corrupt";got,_:=cache.Snapshot("fab");got.Assignments["a"]="changed";got.Order[0]="changed";again,_:=cache.Snapshot("fab");if again.Assignments["a"]!="t1"||again.Order[0]!="a"{t.Fatalf("cache polluted: %#v",again)};var wg sync.WaitGroup;for i:=0;i<20;i++{wg.Add(1);go func(v int64){defer wg.Done();p:=BuildPlan(input);p.Version=v;cache.Publish(p);_,_=cache.Snapshot("fab")}(int64(i))};wg.Wait()}
''')
# 8 reservations
w('internal/reservations/saga_store.go',r'''package reservations
import("context";"sync";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ResourceLease struct{Resource,Owner string;Generation int64}
type SagaStore struct{mu sync.Mutex;owners map[string]ResourceLease;generation int64}
func NewSagaStore()*SagaStore{return &SagaStore{owners:map[string]ResourceLease{}}}
func(s *SagaStore)Acquire(ctx context.Context,resource,owner string)(ResourceLease,error){if err:=ctx.Err();err!=nil{return ResourceLease{},err};s.mu.Lock();defer s.mu.Unlock();if _,ok:=s.owners[resource];ok{return ResourceLease{},platform.ErrConflict};s.generation++;x:=ResourceLease{Resource:resource,Owner:owner,Generation:s.generation};s.owners[resource]=x;return x,nil}
func(s *SagaStore)Release(ctx context.Context,x ResourceLease)error{if err:=ctx.Err();err!=nil{return err};s.mu.Lock();defer s.mu.Unlock();current,ok:=s.owners[x.Resource];if !ok{return nil};if current.Owner!=x.Owner||current.Generation!=x.Generation{return platform.ErrConflict};delete(s.owners,x.Resource);return nil}
func(s *SagaStore)Owner(resource string)string{s.mu.Lock();defer s.mu.Unlock();return s.owners[resource].Owner}
''')
w('internal/reservations/saga.go',r'''package reservations
import("context";"errors";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type ReservationSaga struct{store *SagaStore;events platform.EventSink;clock platform.Clock}
func NewReservationSaga(s *SagaStore,e platform.EventSink,c platform.Clock)*ReservationSaga{return &ReservationSaga{store:s,events:e,clock:c}}
func(s *ReservationSaga)Execute(ctx context.Context,owner string,resources []string)([]ResourceLease,error){leases:=make([]ResourceLease,0,len(resources));for _,resource:=range resources{x,err:=s.store.Acquire(ctx,resource,owner);if err!=nil{return nil,errors.Join(platform.Wrap("acquire","reservation",resource,err),s.compensate(leases))};leases=append(leases,x);if err=s.events.Publish(ctx,platform.Event{Topic:"reservations.resource-acquired",Key:resource,Version:x.Generation,At:s.clock.Now(),Attributes:map[string]string{"owner":owner}});err!=nil{return nil,errors.Join(platform.Wrap("publish","reservation",resource,err),s.compensate(leases))}};return leases,nil}
func(s *ReservationSaga)compensate(leases []ResourceLease)error{ctx,cancel:=context.WithTimeout(context.Background(),time.Second);defer cancel();var joined error;for i:=len(leases)-1;i>=0;i--{if err:=s.store.Release(ctx,leases[i]);err!=nil{joined=errors.Join(joined,err)}};return joined}
''')
w('internal/reservations/saga_test.go',r'''package reservations
import("context";"errors";"testing";"time";"semiconductor-fab-lot-dispatch-service/internal/platform")
type nthFailSink struct{n int};func(s *nthFailSink)Publish(context.Context,platform.Event)error{s.n--;if s.n==0{return errors.New("event failure")};return nil}
func TestReservationSagaCompensatesEveryAcquiredResource(t *testing.T){store:=NewSagaStore();saga:=NewReservationSaga(store,&nthFailSink{n:3},platform.NewManualClock(time.Now()));_,err:=saga.Execute(context.Background(),"lot-a",[]string{"tool","reticle","carrier","transport"});if err==nil{t.Fatal("expected failure")};for _,resource:=range []string{"tool","reticle","carrier","transport"}{if owner:=store.Owner(resource);owner!=""{t.Fatalf("leaked %s to %s",resource,owner)}}}
''')
# 9 qualifications
w('internal/qualifications/probe_set.go',r'''package qualifications
import"sync"
type ProbeResult struct{Probe string;Passed bool;Detail string}
type ProbeSet struct{mu sync.Mutex;results map[string]ProbeResult}
func NewProbeSet()*ProbeSet{return &ProbeSet{results:map[string]ProbeResult{}}}
func(s *ProbeSet)Record(r ProbeResult){s.mu.Lock();s.results[r.Probe]=r;s.mu.Unlock()}
func(s *ProbeSet)Snapshot()[]ProbeResult{s.mu.Lock();defer s.mu.Unlock();out:=make([]ProbeResult,0,len(s.results));for _,r:=range s.results{out=append(out,r)};return out}
''')
w('internal/qualifications/probe_runner.go',r'''package qualifications
import("context";"sync")
type Probe func(context.Context)(ProbeResult,error)
type ProbeRunner struct{Parallelism int}
func(r ProbeRunner)Run(ctx context.Context,probes map[string]Probe)([]ProbeResult,error){limit:=r.Parallelism;if limit<1{limit=1};runCtx,cancel:=context.WithCancel(ctx);defer cancel();names:=make(chan string);set:=NewProbeSet();errCh:=make(chan error,1);var wg sync.WaitGroup;for i:=0;i<limit;i++{wg.Add(1);go func(){defer wg.Done();for{select{case<-runCtx.Done():return;case name,ok:=<-names:if !ok{return};result,err:=probes[name](runCtx);if err!=nil{select{case errCh<-err:default:};cancel();return};result.Probe=name;set.Record(result)}}}()};go func(){defer close(names);for name:=range probes{select{case<-runCtx.Done():return;case names<-name:}}}();wg.Wait();select{case err:=<-errCh:return set.Snapshot(),err;default:};if err:=ctx.Err();err!=nil{return set.Snapshot(),err};return set.Snapshot(),nil}
''')
w('internal/qualifications/probe_runner_test.go',r'''package qualifications
import("context";"errors";"sync/atomic";"testing";"time")
func TestProbeRunnerCancelsAndJoinsEveryProbe(t *testing.T){ctx,cancel:=context.WithCancel(context.Background());var active atomic.Int32;started:=make(chan struct{},3);probe:=func(ctx context.Context)(ProbeResult,error){active.Add(1);defer active.Add(-1);started<-struct{}{};<-ctx.Done();return ProbeResult{},ctx.Err()};done:=make(chan error,1);go func(){_,err:=ProbeRunner{Parallelism:3}.Run(ctx,map[string]Probe{"a":probe,"b":probe,"c":probe});done<-err}();for i:=0;i<3;i++{<-started};cancel();select{case err:=<-done:if !errors.Is(err,context.Canceled){t.Fatalf("err=%v",err)};case<-time.After(time.Second):t.Fatal("runner stuck")};if active.Load()!=0{t.Fatalf("active=%d",active.Load())}}
''')
# 10 batching
w('internal/batching/wafer_set.go',r'''package batching
type WaferLot struct{ID,Recipe,Zone string;Wafers []int;Attributes map[string]string;Priority int}
func(x WaferLot)Clone()WaferLot{x.Wafers=append([]int(nil),x.Wafers...);x.Attributes=cloneAttrs(x.Attributes);return x}
func cloneAttrs(in map[string]string)map[string]string{out:=make(map[string]string,len(in));for k,v:=range in{out[k]=v};return out}
type AssembledBatch struct{Recipe,Zone string;Lots []WaferLot;WaferCount int}
func(x AssembledBatch)Clone()AssembledBatch{out:=x;out.Lots=make([]WaferLot,len(x.Lots));for i,lot:=range x.Lots{out.Lots[i]=lot.Clone()};return out}
''')
w('internal/batching/assembler.go',r'''package batching
import"sort"
type Assembler struct{MaxWafers int;MaxLots int}
func(a Assembler)Assemble(input []WaferLot,recipe,zone string)AssembledBatch{candidates:=make([]WaferLot,0,len(input));for _,lot:=range input{if lot.Recipe==recipe&&lot.Zone==zone{candidates=append(candidates,lot.Clone())}};sort.SliceStable(candidates,func(i,j int)bool{if candidates[i].Priority==candidates[j].Priority{return candidates[i].ID<candidates[j].ID};return candidates[i].Priority>candidates[j].Priority});out:=AssembledBatch{Recipe:recipe,Zone:zone,Lots:make([]WaferLot,0,a.MaxLots)};for _,lot:=range candidates{if len(out.Lots)>=a.MaxLots{break};if out.WaferCount+len(lot.Wafers)>a.MaxWafers{continue};out.Lots=append(out.Lots,lot.Clone());out.WaferCount+=len(lot.Wafers)};return out}
''')
w('internal/batching/assembler_test.go',r'''package batching
import("reflect";"testing")
func TestAssemblerDoesNotMutateOrAliasInputLots(t *testing.T){input:=[]WaferLot{{ID:"low",Recipe:"r",Zone:"z",Priority:1,Wafers:[]int{1,2},Attributes:map[string]string{"owner":"a"}},{ID:"high",Recipe:"r",Zone:"z",Priority:9,Wafers:[]int{3,4},Attributes:map[string]string{"owner":"b"}}};before:=[]string{input[0].ID,input[1].ID};batch:=Assembler{MaxWafers:10,MaxLots:2}.Assemble(input,"r","z");if !reflect.DeepEqual(before,[]string{input[0].ID,input[1].ID}){t.Fatal("input reordered")};batch.Lots[0].Wafers[0]=99;batch.Lots[0].Attributes["owner"]="changed";if input[1].Wafers[0]!=3||input[1].Attributes["owner"]!="b"{t.Fatalf("input aliased: %#v",input[1])}}
''')
