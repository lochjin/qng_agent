const hre = require("hardhat");
const fs = require('fs');

async function main() {
  console.log("🚀 开始部署 MTK 质押合约...");

  // 获取部署者账户
  const [deployer] = await hre.ethers.getSigners();
  console.log("📋 部署账户:", deployer.address);
  
  // 检查 ethers 版本并使用正确的 API
  const balance = await deployer.provider.getBalance(deployer.address);
  console.log("💰 账户余额:", hre.ethers.formatEther(balance));

  // 读取已部署的合约地址
  let deployedContracts = {};
  try {
    const deployedData = fs.readFileSync('deployed.json', 'utf8');
    deployedContracts = JSON.parse(deployedData);
  } catch (error) {
    console.log("⚠️ 未找到 deployed.json 文件，将创建新文件");
  }

  // 检查 MTK 代币合约地址
  if (!deployedContracts.MyToken) {
    throw new Error("❌ 请先部署 MyToken 合约！");
  }

  const mtkTokenAddress = deployedContracts.MyToken;
  console.log("🪙 MTK 代币地址:", mtkTokenAddress);

  // 部署 MTKStaking 合约
  console.log("\n📦 部署 MTKStaking 合约...");
  const MTKStaking = await hre.ethers.getContractFactory("MTKStaking");
  const mtkStaking = await MTKStaking.deploy(mtkTokenAddress);

  await mtkStaking.waitForDeployment();
  console.log("✅ MTKStaking 合约部署成功!");
  console.log("📍 合约地址:", mtkStaking.target);

  // 验证合约部署
  console.log("\n🔍 验证合约部署...");
  const stakingToken = await mtkStaking.stakingToken();
  console.log("✅ 质押代币地址:", stakingToken);
  console.log("✅ 最小质押数量:", hre.ethers.formatEther(await mtkStaking.minStakeAmount()));
  console.log("✅ 奖励率:", (await mtkStaking.rewardRate()).toString());

  // 为质押合约充值奖励代币
  console.log("\n💰 为质押合约充值奖励代币...");
  const MyToken = await hre.ethers.getContractFactory("MyToken");
  const mtkToken = MyToken.attach(mtkTokenAddress);
  
  // 检查部署者的 MTK 余额
  const deployerBalance = await mtkToken.balanceOf(deployer.address);
  console.log("📋 部署者 MTK 余额:", hre.ethers.formatEther(deployerBalance));

  if (deployerBalance > 0) {
    // 转移一些代币到质押合约作为奖励池
    const rewardAmount = hre.ethers.parseEther("10000"); // 10,000 MTK
    if (deployerBalance >= rewardAmount) {
      console.log("📤 转移奖励代币到质押合约...");
      const transferTx = await mtkToken.transfer(mtkStaking.target, rewardAmount);
      await transferTx.wait();
      console.log("✅ 已转移", hre.ethers.formatEther(rewardAmount), "MTK 到质押合约");
    } else {
      console.log("⚠️ 余额不足，跳过奖励代币转移");
    }
  }

  // 更新 deployed.json 文件
  deployedContracts.MTKStaking = mtkStaking.target;
  fs.writeFileSync('deployed.json', JSON.stringify(deployedContracts, null, 4));
  console.log("📄 合约地址已保存到 deployed.json");

  // 显示部署摘要
  console.log("\n🎉 部署完成!");
  console.log("=" .repeat(50));
  console.log("📋 部署摘要:");
  console.log("  • MTK Token:", mtkTokenAddress);
  console.log("  • MTK Staking:", mtkStaking.target);
  console.log("  • 网络:", hre.network.name);
  console.log("  • Gas 使用:", "待确认");
  console.log("=" .repeat(50));

  console.log("\n🔧 下一步操作:");
  console.log("1. 更新 config/contracts.json 配置文件");
  console.log("2. 在前端测试质押功能");
  console.log("3. 为用户提供 MTK 代币进行测试");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("❌ 部署失败:", error);
    process.exit(1);
  }); 