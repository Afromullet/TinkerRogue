__________________________________________

# # # **IMPORTANT** # # # 

- Remove a lot of the commands and GUI elements related to the commands. Those are not needed. After this, review all of the command

- Remove Unused GUI elements and GUI elements you ddont want

- Look at the combat and squad services to see which ones are necessary

- Clean up ECSUtil

- Update Unit Tests if Needed


________________________________________________
# Testing 

**Combat Testing**

- Confirm that Targeting Works Correctly in Combat. Such as a unit correctly attacking rows
- Ensure that ranged attacks work correctly. Only units in range should attack. If it does work, make sure squads
can't attack twice. Once with ranged units and once with melee


# Cleanup
-There is both a config package and config.go in game_main package. Fix that
- ecsutil.go has redundant functions
- tilebatch.go and tilerenderer.go use magic numbers in NewTileBatch
- Completely replace deprecated functions
- Remove Quality
- ecsutil.go seems to have redundancies/duplicate functionality. Look at that. 
- Get Rid of common.Monsters. All the other unit related tags shoudl take care of it. 
- Remove UserMessage component. Not needed
- Remove Quality. DrawableShape Quality should be callled something different
- func CursorPosition(playerPos coords.LogicalPosition) (int, int) in graphictypes requires a change. Leftover from how throwables are handled. 
- CombatController has a lot of artifacts of the old roguelike prior to the change. Specifically in regards to throwables. Find a way tohandle that. 
- Input Package has a lot of leftovers from the old roguelike
- Sprite, Tile, and Rendering Batches allocates default sizes for the slices. Look at those. Determine how many we need. Have a larger default size if needed (i.e, NewSpriteBatch)



# Bug Fixes
- Units of the same faction should not be able to occupy the same square
- Throwables are completely broken. I can no longer see the throwable AoE


# Review



- Identify where there is a possiblity of cache invalidation errors. We started to cache things used for ECS query, so we need to 
- Look at how Tile, sprite, and rendering batching truly works
  * Look at rendering package, 
- Review global position system
- The Inventory is a leftover prior to shifting the game to a turn based squad tactics game from a regular roguelike. Think of what you want to do with the inventory
- Review Squad, combat, and worldmap packages. 
- Review all the caches





# Additions 

- Add proper squad/unit cleanup on death
- Add Abilities
- Better Combat Log




# Redundancies?





- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem. Those are just for debugging. Need to handle differnetly

