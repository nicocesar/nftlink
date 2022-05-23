import React, { useState } from "react";

const AuthContext = React.createContext({
  token: "",
  uuid: "",
  setUuid: (value: string) => {},
  getUuid: () => {},
  metamaskAppDeepLink: "",
  connectWallet: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {},
  connectWalletMobile: () => {},
  onGetConnectedWalletAddress: () => {},
  onUpdateWalletAddress: (value:string) => {},
  onLogin: () => { },
  onLogout: () => { },
});

const fakeAuth = () =>
    new Promise<string>((resolve) => {
        setTimeout(() => resolve('2342f2f1d131rf12'), 250);
    });

function isMobileDevice() {
  return 'ontouchstart' in window || 'onmsgesturechange' in window;
}
interface Props {
  children?: JSX.Element | JSX.Element[];
}

const useAuth = () => {
  return React.useContext(AuthContext);
};

const AuthProvider = ({ children }: Props) => {
    const [token, setToken] = useState<string>("")
    const [uuid, setUuid] = useState<string>("")
    const [currentAccount, setCurrentAccount] = useState("");
    const dappUrl = "nftlink-mzlvbqxo4a-uc.a.run.app/"; // TODO enter your dapp URL. For example: https://uniswap.exchange. (don't enter the "https://")
    const metamaskAppDeepLink = "https://metamask.app.link/dapp/" + dappUrl;

    const handleLogin = async () => {
        const token = await fakeAuth();
        setToken(token);
    };

    const handleLogout = () => {
        setToken("");
    };

    const updateWalletAddress = async (value:string) =>{
        setCurrentAccount(value);
    }

    const getConnectedWalletAddress = async () => {
        return currentAccount;
    }

    const getUuid = async () => {
        return uuid;
    }

    const connectWalletMobile = async () => {
      if (!window.ethereum) {
        alert("Get MetaMask!");
        return;
      }

      const accounts = await window.ethereum.request({
        method: "eth_requestAccounts",
      });

      setCurrentAccount(accounts[0]);
    }

    const connectWallet = async (event: React.MouseEvent<HTMLButtonElement>) => {
        try {
          const { ethereum } = window;

          if (!ethereum) {
            alert("Get MetaMask!");
            return;
          }

          const accounts = await ethereum.request({ method: "eth_requestAccounts" });

          console.log("Connected", accounts[0]);
          updateWalletAddress(accounts[0]);
          if (isMobileDevice()) {
            await connectWalletMobile();
          }
        } catch (error) {
          console.log(error)
        }
      }

    const value = {
        token,
        uuid,
        setUuid,
        getUuid,
        metamaskAppDeepLink,
        connectWallet,
        connectWalletMobile,
        onGetConnectedWalletAddress : getConnectedWalletAddress,
        onUpdateWalletAddress : updateWalletAddress,
        onLogin: handleLogin,
        onLogout: handleLogout,
    };



    return (
        <AuthContext.Provider value={value}>
            {children}
        </AuthContext.Provider>
    );

}

export {AuthProvider, useAuth, isMobileDevice};