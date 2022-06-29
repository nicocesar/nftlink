<script lang="ts">
    import { page } from '$app/stores';
    import { browser } from '$app/env';
   import { defaultEvmStores, connected, chainId, allChainsData, signer, signerAddress } from 'svelte-ethers-store';
   import { ethers } from 'ethers';

    let uuid: string = $page?.params['id'];


   let askToInstallMetamask = false;
   // This will only render client-side if the browser is available.
   if(browser) {
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

<!-- backend call for minting -->
<!-- https://nftlink-mzlvbqxo4a-uc.a.run.app/mint/5a84jjJkwz/0x007a45d4898593720B8d4cF796228b0EeFb70Ac0 -->

<svelte:head>
  <title>Welcome!</title>
</svelte:head>


<h1>Redeem an NFT for free!</h1>
<h4>yeah! we even pay the gas for you!</h4>

{#if askToInstallMetamask}
    <p>Please install Metamask to use this app.</p>
{:else}
    {#if !$connected}

    <p>Waiting for the wallet to connect...</p>

    {:else}

    <p>Connected  {$signerAddress} to chain with id {$chainId}</p>
    <!-- TODO: make a check if chainId is supported, also mint might need an extra param -->
    {#await $signer.signMessage( `Sign this to claim an NFT id: ${uuid}` ) then value}
       {#if ethers.utils.verifyMessage( `Sign this to claim an NFT id: ${uuid}`, value ) == $signerAddress }
          Thanks for signing!

        <!-- todo fetch backend  https://nftlink-mzlvbqxo4a-uc.a.run.app/mint/{uuid}/{$signerAddress} -->
          {#await mint(uuid, $signerAddress)}
           Sending to backend...
           {:then value}
              <p>Sucess!!  {value}</p>
           {:catch error}
              <p style="color: red">{error.message}</p>
           {/await}
        {:else}
          The user did not sign!
       {/if}
    {:catch error}
    	<p style="color: red">{error.message}</p>
    {/await}

    {/if}

{/if}