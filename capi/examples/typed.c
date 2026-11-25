#include "fzfp.h"
#include <assert.h>
#include <pthread.h>
#include <stdio.h>
#include <string.h>

static uint64_t shared_handle;
static void *query_thread(void *unused) {
    (void)unused;
    fzfp_find_options opts={2,sizeof(opts),1,1,0};
    for(int i=0;i<100;i++) {
        fzfp_results *out=NULL;
        int status=fzfp_index_find(shared_handle,"zg",2,&opts,&out);
        assert(status==0 || status==3);
        if(status==0) { assert(out && out->count==1); fzfp_results_free(out); }
    }
    return NULL;
}
int main(void) {
    assert(fzfp_abi_version()==1 && fzfp_typed_abi_version()==2);
    char bytes[]="中国\0北京";
    uint64_t offsets[]={0,6,sizeof(bytes)-1};
    fzfp_options opts={2,sizeof(opts),0,0,1,4,0,0};
    uint64_t h=0;
    assert(fzfp_index_create(bytes,sizeof(bytes)-1,offsets,2,&opts,&h)==0 && h);
    memset(bytes,'x',sizeof(bytes)); // construction copied caller buffers
    fzfp_find_options find={2,sizeof(find),50,1,0};
    fzfp_results *r=NULL;
    assert(fzfp_index_find(h,"zg",2,&find,&r)==0);
    assert(r->abi_version==2 && r->count==1 && r->hits[0].index==0);
    assert(r->hits[0].start==0 && r->hits[0].end==2 && r->position_count==2);
    assert(r->positions[0]==0 && r->positions[1]==1);
    fzfp_results_free(r);
    find.limit=-2;
    assert(fzfp_index_find(h,"zg",2,&find,&r)==1 && r==NULL);
    find.limit=1; find.reserved=1;
    assert(fzfp_index_find(h,"zg",2,&find,&r)==1);
    find.reserved=0;
    assert(fzfp_index_find(h,"\xff",1,&find,&r)==1);
    assert(fzfp_index_find(h,NULL,1,&find,&r)==1);
    opts.struct_size=0; uint64_t invalid=1;
    assert(fzfp_index_create(NULL,0,offsets,0,&opts,&invalid)==1 && invalid==0);
    opts.struct_size=sizeof(opts); offsets[1]=999;
    assert(fzfp_index_create(bytes,sizeof(bytes)-1,offsets,2,&opts,&invalid)==1);
    shared_handle=h;
    pthread_t threads[4];
    for(int i=0;i<4;i++) assert(pthread_create(&threads[i],NULL,query_thread,NULL)==0);
    assert(fzfp_index_destroy(h)==0);
    for(int i=0;i<4;i++) assert(pthread_join(threads[i],NULL)==0);
    assert(fzfp_index_destroy(h)==3);
    assert(fzfp_index_find(h,"zg",2,&find,&r)==3 && r==NULL);
    fzfp_results_free(NULL);
    puts("typed ABI ownership/validation/concurrent destroy passed");
}
