/** @type {import('./__types/[id]').RequestHandler} */

import type { RequestEvent } from "@sveltejs/kit";

// This endpoint is a MOCKUP endpoint for the minting of tokens.
// in production this will go to the backend and mint the tokens for the user.

export async function get( e: RequestEvent) {
    if (e.params) {
        return {
          status : 200,
          headers : { 'Content-Type' : 'text/json' }, 
          body:  `{'minted':'something','address':'${e.params.address}','id':'${e.params['id']}'}` }
        };
      };
      