import React from "react";
import { useAuth } from "src/components/AuthProvider";

const ConnectWallet = async (event: React.MouseEvent<HTMLButtonElement>) => {
    const { getConnectedWalletAddress, updateWalletAddress } = useAuth();
    try {
      const { ethereum } = window;

      if (!ethereum) {
        alert("Get MetaMask!");
        return;
      }

      const accounts = await ethereum.request({ method: "eth_requestAccounts" });

      console.log("Connected", accounts[0]);
      updateWalletAddress(accounts[0]);
    } catch (error) {
      console.log(error)
    }
  }

export default ConnectWallet;