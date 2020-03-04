/*
==================================================================================
  Copyright (c) 2020 AT&T Intellectual Property.
  Copyright (c) 2020 Nokia

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
==================================================================================
*/

#include <string.h>
#include "utils.h"

void * rmrInit(void) {
    void* mrc;  // msg router context

    if( (mrc = rmr_init("tcp:4588", 1024, RMRFL_NONE)) == NULL ) {
        fprintf(stderr, "Unable to initialize RMR\n");
        return NULL;
    }

    // Must have a route table before we can send, so wait til RMR is ready
    while(!rmr_ready(mrc)) {
        fprintf(stderr, "Waiting for RMR to be ready ...\n");
        sleep(1);
    }
    fprintf(stderr, "RMR is ready now ...\n");

    return mrc;
}

int rmrSend(void *mrc, int mtype, void *payload, int payload_len, char *meid) {
    int retry_count = 0;
    rmr_mbuf_t *sbuf = rmr_alloc_msg(mrc, 1024);

    sbuf->mtype = mtype;
    sbuf->sub_id = RMR_VOID_SUBID;
    sbuf->state = 0;
    sbuf->len = payload_len;
    memcpy(sbuf->payload, payload, payload_len);
    rmr_str2meid(sbuf, meid);

    do {
        sbuf = rmr_send_msg(mrc, sbuf);
        if (sbuf == NULL) {
            return -1;
        }

        if (sbuf->state == RMR_OK) {
            break;
        }
    } while(sbuf->state == RMR_ERR_RETRY && ++retry_count < 10);

    return sbuf->state;
}

rmr_mbuf_t * rmrRcv(void *mrc) {
    while(1) {
        rmr_mbuf_t *rbuf = rmr_rcv_msg(mrc, NULL);
        if (rbuf != NULL && rbuf->state == RMR_OK) {
            return rbuf;
        }
    }

    return NULL;
}

