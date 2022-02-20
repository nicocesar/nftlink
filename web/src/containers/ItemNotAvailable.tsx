import React from "react";

const ItemNotAvailable: React.FC = (): JSX.Element => {

  return (
    <>
    <div className="bg-red-100 text-center rounded-md p-3 flex justify-center">
    <svg
        className="stroke-2 stroke-current text-red-600 text-center h-8 w-8 mr-2 justify-center"
        viewBox="0 0 24 24"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
    >
        <path d="M0 0h24v24H0z" stroke="none" />
        <circle cx="12" cy="12" r="9" />
        <path d="M9 12l2 2 4-4" />
    </svg>

    <div className="text-red-700 text-center justify-center">
        <div className="font-bold text-xl">Redeem Code Not available!</div>
        <div>Check QR code.</div>
    </div>
</div>
    </>
  );
};

export default ItemNotAvailable;
