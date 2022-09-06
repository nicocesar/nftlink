<script lang="ts">
   import { page } from '$app/stores';
   import { browser } from '$app/env';
   import { defaultEvmStores, connected, chainId, allChainsData, signer, signerAddress } from 'svelte-ethers-store';
   import { ethers } from 'ethers';

   let uuid = $page?.params['id'];

   let askToInstallMetamask = false;
   if(browser) { // This will only render client-side if the browser is available.
      defaultEvmStores.setProvider().catch(() => {
         askToInstallMetamask = true;
      });
   }

   async function mint(uuid:string, address:string) {
		const res = await fetch(`/mint/${uuid}/${address}` );
        
		const text = await res.text();

		if (res.ok) {
			return text;
		} else {
			throw new Error(text);
		}
	}
</script>

<svelte:head>
  <title>Claiming {uuid}!</title>
</svelte:head>


<h1>Redeem an NFT for free!</h1>
<h4>yeah! we even pay the gas for you!</h4>

{#if askToInstallMetamask}
    <p id="install-metamask">Please <a href="https://metamask.io/">install Metamask</a> to use this app.</p>
{:else}
    {#if !$connected}

    <p id="waiting-for-wallet">Waiting for the wallet to connect...</p>

    {:else}

    <p>Connected  {$signerAddress} to chain with id {$chainId}</p>
    <!-- TODO: make a check if chainId is supported, also mint might need an extra param -->
    {#await $signer.signMessage( `Sign this to claim an NFT id: ${uuid}` ) then value}
       {#if ethers.utils.verifyMessage( `Sign this to claim an NFT id: ${uuid}`, value ) == $signerAddress }
          Thanks for signing! We are now sending the NFT to your wallet {$signerAddress}
          {#await mint(uuid, $signerAddress)}
           Sending to backend...
           {:then value}
           <!-- {"type":"0x0","nonce":"0xe","gasPrice":"0x6fc23ac00","maxPriorityFeePerGas":null,"maxFeePerGas":null,"gas":"0x493e0","value":"0x0","input":"0xd204c45e000000000000000000000000007a45d4898593720b8d4cf796228b0eefb70ac00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000002e516d5455314166514e6e5850435847316651594e79516d4c6141675631767169784b6669775353547a78416b666f000000000000000000000000000000000000","v":"0x2c","r":"0xa10526f9904afe2392c161d6d2275f0aa3b99e7e711d125deb8892c1ec513660","s":"0x1860f6c7bb622e84c4c30125adc19212e6be5652fbae6fff670ffff3e8ad5a3e","to":"0x6d62f9b5d6ba45261ee1b0fa551c5dd28b3e2881","hash":"0x56fd5dd659e120064a336be0f7fdf69d384522e5fa047433029fb63f1d68ef23"} -->
              <p>Sucess!! This is your receipt: {value}</p>
           {:catch error}
              <p style="color: red">{error.message}</p>
           {/await}
        {:else}
          <p>Sorry but, you need to sign that message to claim the NFT.</p>
       {/if}
    {:catch error}
    	<p style="color: red">{error.message}</p>
    {/await}

    {/if}

{/if}