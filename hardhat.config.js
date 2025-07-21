require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.28",
  networks: {
    customnet: {
      url: "http://47.242.255.132:1234/",
      chainId: 8134,
      accounts: ["0x8fd8603f6b6320026c3a1e30e22d4485352d5d18aa33f77503f3137aea0d59d5"]
    }
  }
};
