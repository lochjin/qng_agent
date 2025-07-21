// scripts/deploy.js
const { ethers } = require("hardhat");

async function main() {
  // 1. 部署 MyToken 合约
  const initialSupply = ethers.parseUnits("1000000", 18); // 100万个代币
  const MyToken = await ethers.getContractFactory("MyToken");
  const myToken = await MyToken.deploy(initialSupply);
  await myToken.waitForDeployment();
  console.log("MyToken 部署地址:", await myToken.getAddress());

  // 2. 部署 SimpleSwap 合约
  const SimpleSwap = await ethers.getContractFactory("SimpleSwap");
  const simpleSwap = await SimpleSwap.deploy(await myToken.getAddress());
  await simpleSwap.waitForDeployment();
  console.log("SimpleSwap 部署地址:", await simpleSwap.getAddress());

  // 3. 给 SimpleSwap 合约充值一些 Token（比如 10万个）
  const depositAmount = ethers.parseUnits("100000", 18);
  const tx = await myToken.transfer(await simpleSwap.getAddress(), depositAmount);
  await tx.wait();
  console.log(`已向 SimpleSwap 合约充值 ${depositAmount.toString()} MTK`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
}); 