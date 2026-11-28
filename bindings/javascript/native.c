#include <node_api.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include "fzfp.h"
_Static_assert(sizeof(fzfp_options)==32, "options layout");
_Static_assert(sizeof(fzfp_find_options)==24, "find options layout");
_Static_assert(sizeof(fzfp_hit)==48, "hit layout");

#define NAPI(call) do { if ((call) != napi_ok) { napi_throw_error(env, NULL, "invalid native argument or allocation failure"); return NULL; } } while(0)
static int check(napi_env env, int status) {
    if (!status) return 1;
    napi_throw_error(env, NULL, status == 1 ? "invalid fzfp input/options" : status == 3 ? "closed or invalid fzfp index" : "fzfp internal/allocation failure");
    return 0;
}
static napi_value number(napi_env env, double n) { napi_value v; if(napi_create_double(env,n,&v)!=napi_ok) return NULL; return v; }
static int integer(napi_env env, napi_value v, double min, double max, double *out) {
    if(napi_get_value_double(env,v,out)!=napi_ok || !isfinite(*out) || floor(*out)!=*out || *out<min || *out>max) {
        napi_throw_range_error(env,NULL,"expected integer in range"); return 0;
    } return 1;
}
static napi_value create(napi_env env, napi_callback_info info) {
    napi_value a[6], value; size_t n=6, bl=0, ol=0; void *b=NULL,*o=NULL; double scheme,cs,norm,workers;
    NAPI(napi_get_cb_info(env,info,&n,a,NULL,NULL));
    if(n!=6) { napi_throw_type_error(env,NULL,"create expects six arguments"); return NULL; }
    NAPI(napi_get_buffer_info(env,a[0],&b,&bl)); NAPI(napi_get_buffer_info(env,a[1],&o,&ol));
    if(ol<8 || ol%8 || ol/8>10000001) { napi_throw_range_error(env,NULL,"invalid offsets"); return NULL; }
    if(!integer(env,a[2],0,2,&scheme)||!integer(env,a[3],0,1,&cs)||!integer(env,a[4],0,1,&norm)||!integer(env,a[5],0,64,&workers)) return NULL;
    uint64_t *offsets=malloc(ol); if(!offsets) { napi_throw_error(env,NULL,"allocation failed"); return NULL; }
    memcpy(offsets,o,ol);
    fzfp_options opts={2,sizeof(opts),(uint32_t)scheme,(uint32_t)cs,(uint32_t)norm,(uint32_t)workers,0,0};
    uint64_t handle=0; int status=fzfp_index_create(b,bl,offsets,ol/8-1,&opts,&handle); free(offsets);
    if(!check(env,status)) return NULL;
    if(napi_create_bigint_uint64(env,handle,&value)!=napi_ok) { fzfp_index_destroy(handle); return NULL; }
    return value;
}
static int get_handle(napi_env env,napi_value v,uint64_t *h) {
    bool lossless=false;
    if(napi_get_value_bigint_uint64(env,v,h,&lossless)!=napi_ok || !lossless) { napi_throw_type_error(env,NULL,"invalid handle"); return 0; }
    return 1;
}
static napi_value destroy(napi_env env,napi_callback_info info) {
    size_t n=1; napi_value a[1],v; uint64_t h;
    NAPI(napi_get_cb_info(env,info,&n,a,NULL,NULL));
    if(n!=1 || !get_handle(env,a[0],&h)) return NULL;
    if(!check(env,fzfp_index_destroy(h))) return NULL;
    NAPI(napi_get_undefined(env,&v)); return v;
}
// NAPI failures while materializing results always free the C allocation.
#define OUT(call) do { if((call)!=napi_ok) { fzfp_results_free(r); napi_throw_error(env,NULL,"result materialization failed"); return NULL; } } while(0)
static napi_value find(napi_env env,napi_callback_info info) {
    size_t n=4,ql=0; napi_value a[4],result,rows,metrics; uint64_t h; void *q=NULL; double limit,flags;
    NAPI(napi_get_cb_info(env,info,&n,a,NULL,NULL));
    if(n!=4 || !get_handle(env,a[0],&h)) return NULL;
    NAPI(napi_get_buffer_info(env,a[1],&q,&ql));
    if(!integer(env,a[2],-1,9007199254740991.0,&limit)||!integer(env,a[3],0,7,&flags)) return NULL;
    fzfp_find_options opts={2,sizeof(opts),(int64_t)limit,(uint32_t)flags,0}; fzfp_results *r=NULL;
    if(!check(env,fzfp_index_find(h,q,ql,&opts,&r))) return NULL;
    OUT(napi_create_object(env,&result)); OUT(napi_create_array_with_length(env,r->count,&rows));
    for(uint64_t i=0;i<r->count;i++) {
        fzfp_hit hit=r->hits[i]; napi_value row,positions;
        OUT(napi_create_array_with_length(env,5,&row));
        OUT(napi_set_element(env,row,0,number(env,hit.index))); OUT(napi_set_element(env,row,1,number(env,hit.score)));
        OUT(napi_set_element(env,row,2,number(env,hit.start))); OUT(napi_set_element(env,row,3,number(env,hit.end)));
        OUT(napi_create_array_with_length(env,hit.position_count,&positions));
        for(uint64_t j=0;j<hit.position_count;j++) OUT(napi_set_element(env,positions,j,number(env,r->positions[hit.position_offset+j])));
        OUT(napi_set_element(env,row,4,positions)); OUT(napi_set_element(env,rows,i,row));
    }
    OUT(napi_create_object(env,&metrics));
    const char *names[]={"prepare","scan","select","trace","copy","native"};
    double values[]={r->prepare_ms,r->scan_ms,r->select_ms,r->trace_ms,r->copy_ms,r->native_ms};
    for(int i=0;i<6;i++) OUT(napi_set_named_property(env,metrics,names[i],number(env,values[i])));
    OUT(napi_set_named_property(env,result,"rows",rows)); OUT(napi_set_named_property(env,result,"timings",metrics));
    fzfp_results_free(r); return result;
}
static napi_value json_call(napi_env env,napi_callback_info info) {
    size_t n=1,len=0,size=0; napi_value a[1],v; void *b; int32_t status;
    NAPI(napi_get_cb_info(env,info,&n,a,NULL,NULL)); if(n!=1) return NULL;
    NAPI(napi_get_buffer_info(env,a[0],&b,&len));
    char *output=fzfp_call(b,len,&size,&status); if(!output) { napi_throw_error(env,NULL,"null JSON result"); return NULL; }
    napi_status s=napi_create_string_utf8(env,output,size,&v); fzfp_free(output);
    if(s!=napi_ok) return NULL; return v;
}
static napi_value init(napi_env env,napi_value exports) {
    if(fzfp_typed_abi_version()!=2) { napi_throw_error(env,NULL,"ABI mismatch"); return NULL; }
    napi_property_descriptor props[]={
        {"create",NULL,create,NULL,NULL,NULL,napi_default,NULL},
        {"find",NULL,find,NULL,NULL,NULL,napi_default,NULL},
        {"destroy",NULL,destroy,NULL,NULL,NULL,napi_default,NULL},
        {"json",NULL,json_call,NULL,NULL,NULL,napi_default,NULL}};
    NAPI(napi_define_properties(env,exports,4,props)); return exports;
}
NAPI_MODULE(NODE_GYP_MODULE_NAME,init)
