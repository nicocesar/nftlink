import React, { useEffect, useState } from "react";
import {useNavigate} from 'react-router-dom';
import ItemAvailable from "./ItemAvailable";
import ItemNotAvailable from "./ItemNotAvailable";

const Welcome: React.FC = (): JSX.Element => {
  const params = new URLSearchParams(window.location.search);
  const paramValue = params.get("uuid");
  const [response, setResponse] = useState<number>();
  const navigate = useNavigate();

  function timeout(delay: number) {
    return new Promise( res => setTimeout(res, delay) );
  }

  const goHome = async () => {
    navigate('/home')
  };

  useEffect(() => {
    const checkCodeAvailavility = async () => {
      const localResponse = await fetch('https://nftlink-mzlvbqxo4a-uc.a.run.app/check/'+paramValue);
      setResponse(localResponse.status);
    }
    checkCodeAvailavility();
  }, [paramValue])

  return (
    <>
     <div className="relative bg-white overflow-hidden">
      <div className="max-w-7xl mx-auto">
        <div className="relative z-10 pb-8 bg-white sm:pb-16 md:pb-20 lg:w-full lg:w-full lg:pb-28 xl:pb-32">
          <main className="mt-10 mx-auto max-w-full px-4 sm:mt-12 sm:px-6 md:mt-16 lg:mt-20 lg:px-8 xl:mt-28">
            <div className="sm:text-center lg:text-center">
              <h1 className="text-4xl tracking-tight font-extrabold text-gray-900 sm:text-5xl md:text-6xl">
                <span className="block xl:inline">Grow your business with</span>{' '}
                <span className="block text-indigo-600 xl:inline">NFT</span>{' '}
                <span className="block xl:inline">techology</span>{' '}
              </h1>
              <p className="mt-3 text-base text-gray-500 sm:mt-5 sm:text-lg sm:max-w-full sm:mx-auto md:mt-5 md:text-xl lg:mx-0">
                Checking redeem code
              </p>

              {response === 200 &&
              <>
                <ItemAvailable/>
                <button onClick={goHome} className="w-full flex items-center justify-center px-8 py-3 border border-transparent text-base font-medium rounded-md text-indigo-700 bg-indigo-100 hover:bg-indigo-200 md:py-4 md:text-lg md:px-10" name="button 1">
                  Go home page
                </button>
              </>
              }
              {response === 201 &&
                <ItemNotAvailable/>
              }
            </div>
          </main>
        </div>
      </div>
    </div>
    </>
  );
};

export default Welcome;